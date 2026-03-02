package tg

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"mime"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gotd/td/telegram/uploader"
	tdtg "github.com/gotd/td/tg"
	"golang.org/x/sync/errgroup"
)

// ResolveTarget resolves a target string to a ResolvedTarget.
// Supports: "me", "@username", "username", numeric channel/user IDs.
func (c *GotdClient) ResolveTarget(ctx context.Context, target string) (ResolvedTarget, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		target = "me"
	}

	if strings.EqualFold(target, "me") {
		self, err := c.telegram.Self(ctx)
		if err != nil {
			return ResolvedTarget{}, mapGotdError(err)
		}
		peer := &tdtg.InputPeerUser{
			UserID:     self.ID,
			AccessHash: self.AccessHash,
		}
		c.cachePeer("user", self.ID, peer)
		return ResolvedTarget{Kind: "user", ID: self.ID, Raw: target}, nil
	}

	// Try numeric ID.
	if id, err := strconv.ParseInt(target, 10, 64); err == nil {
		return c.resolveNumericID(ctx, id, target)
	}

	// Username (with or without @).
	username := strings.TrimPrefix(target, "@")
	return c.resolveUsername(ctx, username, target)
}

func (c *GotdClient) resolveNumericID(_ context.Context, id int64, raw string) (ResolvedTarget, error) {
	if id < 0 {
		idStr := strconv.FormatInt(id, 10)
		// IDs starting with -100 are channels/supergroups.
		if strings.HasPrefix(idStr, "-100") {
			channelID := -id - 1000000000000
			peer := &tdtg.InputPeerChannel{ChannelID: channelID}
			c.cachePeer("channel", channelID, peer)
			return ResolvedTarget{Kind: "channel", ID: channelID, Raw: raw}, nil
		}
		// Other negative IDs are regular group chats.
		chatID := -id
		peer := &tdtg.InputPeerChat{ChatID: chatID}
		c.cachePeer("chat", chatID, peer)
		return ResolvedTarget{Kind: "chat", ID: chatID, Raw: raw}, nil
	}

	// Positive IDs are users (no access_hash available for bare numeric IDs).
	peer := &tdtg.InputPeerUser{UserID: id}
	c.cachePeer("user", id, peer)
	return ResolvedTarget{Kind: "user", ID: id, Raw: raw}, nil
}

func (c *GotdClient) resolveUsername(ctx context.Context, username, raw string) (ResolvedTarget, error) {
	api, err := c.getAPI()
	if err != nil {
		return ResolvedTarget{}, err
	}

	resolved, err := api.ContactsResolveUsername(ctx, &tdtg.ContactsResolveUsernameRequest{
		Username: username,
	})
	if err != nil {
		return ResolvedTarget{}, mapGotdError(err)
	}

	switch p := resolved.Peer.(type) {
	case *tdtg.PeerUser:
		for _, u := range resolved.Users {
			if user, ok := u.(*tdtg.User); ok && user.ID == p.UserID {
				peer := &tdtg.InputPeerUser{
					UserID:     user.ID,
					AccessHash: user.AccessHash,
				}
				c.cachePeer("user", user.ID, peer)
				return ResolvedTarget{Kind: "user", ID: user.ID, Raw: raw}, nil
			}
		}
	case *tdtg.PeerChat:
		peer := &tdtg.InputPeerChat{ChatID: p.ChatID}
		c.cachePeer("chat", p.ChatID, peer)
		return ResolvedTarget{Kind: "chat", ID: p.ChatID, Raw: raw}, nil
	case *tdtg.PeerChannel:
		for _, ch := range resolved.Chats {
			if channel, ok := ch.(*tdtg.Channel); ok && channel.ID == p.ChannelID {
				peer := &tdtg.InputPeerChannel{
					ChannelID:  channel.ID,
					AccessHash: channel.AccessHash,
				}
				c.cachePeer("channel", channel.ID, peer)
				return ResolvedTarget{Kind: "channel", ID: channel.ID, Raw: raw}, nil
			}
		}
	}

	return ResolvedTarget{}, fmt.Errorf("could not resolve target: %s", raw)
}

// SendSingle sends a single media file.
func (c *GotdClient) SendSingle(ctx context.Context, req SendSingleRequest) (SendResult, error) {
	peer, err := c.getPeer(req.Target)
	if err != nil {
		return SendResult{}, err
	}

	api, err := c.getAPI()
	if err != nil {
		return SendResult{}, err
	}

	// Create a fresh uploader per request — WithProgress mutates in place.
	up, err := c.newUploader()
	if err != nil {
		return SendResult{}, err
	}
	if req.Progress != nil {
		up = up.WithProgress(&progressAdapter{fn: req.Progress})
	}

	file, err := up.FromPath(ctx, req.Path)
	if err != nil {
		return SendResult{}, mapGotdError(err)
	}

	var thumb tdtg.InputFileClass
	if req.ThumbnailPath != "" {
		thumbUp, err := c.newUploader()
		if err == nil {
			thumb, _ = thumbUp.FromPath(ctx, req.ThumbnailPath) // optional
		}
	}

	media := buildInputMedia(file, req.Path, req.ForceDocument, req.SupportsStreaming, thumb, req.Video)

	updates, err := api.MessagesSendMedia(ctx, &tdtg.MessagesSendMediaRequest{
		Peer:     peer,
		Media:    media,
		Message:  req.Caption,
		RandomID: cryptoRandInt63(),
	})
	if err != nil {
		return SendResult{}, mapGotdError(err)
	}

	return extractSendResult(updates), nil
}

// SendAlbum sends multiple media files as a grouped album.
// Files are uploaded in parallel for better throughput, then sent as one album.
func (c *GotdClient) SendAlbum(ctx context.Context, req SendAlbumRequest) (SendResult, error) {
	if len(req.Items) == 0 {
		return SendResult{}, fmt.Errorf("album must contain at least one item")
	}
	if len(req.Items) > 10 {
		return SendResult{}, fmt.Errorf("album cannot contain more than 10 items (got %d)", len(req.Items))
	}

	peer, err := c.getPeer(req.Target)
	if err != nil {
		return SendResult{}, err
	}

	api, err := c.getAPI()
	if err != nil {
		return SendResult{}, err
	}

	// Upload all files in parallel, each with its own uploader instance.
	// Results are written by index so no mutex is needed.
	results := make([]tdtg.InputSingleMedia, len(req.Items))
	g, gctx := errgroup.WithContext(ctx)

	for i, item := range req.Items {
		g.Go(func() error {
			// Each goroutine gets a fresh uploader (WithProgress mutates).
			up, err := c.newUploader()
			if err != nil {
				return err
			}

			file, err := up.FromPath(gctx, item.Path)
			if err != nil {
				return mapGotdError(err)
			}

			var thumb tdtg.InputFileClass
			if item.ThumbnailPath != "" {
				thumbUp, err := c.newUploader()
				if err == nil {
					thumb, _ = thumbUp.FromPath(gctx, item.ThumbnailPath)
				}
			}

			inputMedia := buildInputMedia(file, item.Path, item.ForceDocument, item.SupportsStreaming, thumb, item.Video)

			// Pre-upload to get server-side media reference.
			uploaded, err := api.MessagesUploadMedia(gctx, &tdtg.MessagesUploadMediaRequest{
				Peer:  peer,
				Media: inputMedia,
			})
			if err != nil {
				return mapGotdError(err)
			}

			refMedia, err := messageMediaToInput(uploaded)
			if err != nil {
				return err
			}

			results[i] = tdtg.InputSingleMedia{
				Media:    refMedia,
				RandomID: cryptoRandInt63(),
				Message:  item.Caption,
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return SendResult{}, err
	}

	updates, err := api.MessagesSendMultiMedia(ctx, &tdtg.MessagesSendMultiMediaRequest{
		Peer:       peer,
		MultiMedia: results,
	})
	if err != nil {
		return SendResult{}, mapGotdError(err)
	}

	return extractSendResult(updates), nil
}

// buildInputMedia creates the appropriate InputMedia for a file.
func buildInputMedia(file tdtg.InputFileClass, path string, forceDocument, supportsStreaming bool, thumb tdtg.InputFileClass, video *VideoMeta) tdtg.InputMediaClass {
	kind := detectFileKind(path)

	if kind == "image" && !forceDocument {
		return &tdtg.InputMediaUploadedPhoto{
			File: file,
		}
	}

	mimeType := detectMIMEType(path)
	attrs := []tdtg.DocumentAttributeClass{
		&tdtg.DocumentAttributeFilename{FileName: filepath.Base(path)},
	}

	if kind == "video" {
		va := &tdtg.DocumentAttributeVideo{
			SupportsStreaming: supportsStreaming,
		}
		if video != nil {
			va.Duration = video.Duration
			va.W = video.Width
			va.H = video.Height
		}
		attrs = append(attrs, va)
	}

	return &tdtg.InputMediaUploadedDocument{
		File:       file,
		MimeType:   mimeType,
		Attributes: attrs,
		Thumb:      thumb,
	}
}

// messageMediaToInput converts an uploaded MessageMedia to an InputMedia reference.
func messageMediaToInput(mm tdtg.MessageMediaClass) (tdtg.InputMediaClass, error) {
	switch m := mm.(type) {
	case *tdtg.MessageMediaPhoto:
		photo, ok := m.Photo.AsNotEmpty()
		if !ok {
			return nil, fmt.Errorf("empty photo in upload response")
		}
		return &tdtg.InputMediaPhoto{
			ID: photo.AsInput(),
		}, nil
	case *tdtg.MessageMediaDocument:
		doc, ok := m.Document.AsNotEmpty()
		if !ok {
			return nil, fmt.Errorf("empty document in upload response")
		}
		return &tdtg.InputMediaDocument{
			ID: doc.AsInput(),
		}, nil
	}
	return nil, fmt.Errorf("unsupported media type for album: %T", mm)
}

// extractSendResult parses message IDs and metadata from the updates response.
func extractSendResult(updates tdtg.UpdatesClass) SendResult {
	var result SendResult

	switch u := updates.(type) {
	case *tdtg.Updates:
		extractMessagesFromUpdates(u.Updates, &result)
	case *tdtg.UpdatesCombined:
		extractMessagesFromUpdates(u.Updates, &result)
	case *tdtg.UpdateShortSentMessage:
		result.MessageIDs = []int{u.ID}
		result.Messages = []SentMessage{{ID: u.ID}}
	}

	return result
}

func extractMessagesFromUpdates(updates []tdtg.UpdateClass, result *SendResult) {
	for _, upd := range updates {
		var msg *tdtg.Message
		switch m := upd.(type) {
		case *tdtg.UpdateNewMessage:
			msg, _ = m.Message.(*tdtg.Message)
		case *tdtg.UpdateNewChannelMessage:
			msg, _ = m.Message.(*tdtg.Message)
		}
		if msg == nil {
			continue
		}
		sm := convertSentMessage(msg)
		result.Messages = append(result.Messages, sm)
		result.MessageIDs = append(result.MessageIDs, msg.ID)
		if msg.GroupedID != 0 && result.GroupID == "" {
			result.GroupID = strconv.FormatInt(msg.GroupedID, 10)
		}
	}
}

func convertSentMessage(msg *tdtg.Message) SentMessage {
	sm := SentMessage{ID: msg.ID}
	if msg.Media == nil {
		return sm
	}

	switch m := msg.Media.(type) {
	case *tdtg.MessageMediaPhoto:
		sm.MediaKind = "photo"
	case *tdtg.MessageMediaDocument:
		if doc, ok := m.Document.AsNotEmpty(); ok {
			sm.Size = doc.Size
			for _, attr := range doc.Attributes {
				switch a := attr.(type) {
				case *tdtg.DocumentAttributeVideo:
					sm.MediaKind = "video"
					sm.Duration = a.Duration
					sm.Width = a.W
					sm.Height = a.H
					sm.SupportsStreaming = a.SupportsStreaming
				case *tdtg.DocumentAttributeFilename:
					sm.FileName = a.FileName
				}
			}
			if sm.MediaKind == "" {
				sm.MediaKind = "document"
			}
		}
	}

	return sm
}

func detectFileKind(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".heic":
		return "image"
	case ".mp4", ".mov", ".mkv", ".webm":
		return "video"
	default:
		return "document"
	}
}

func detectMIMEType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	// Try standard library first.
	if m := mime.TypeByExtension(ext); m != "" {
		return m
	}
	// Fallback for common types.
	switch ext {
	case ".mp4":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	case ".mkv":
		return "video/x-matroska"
	case ".webm":
		return "video/webm"
	case ".heic":
		return "image/heic"
	default:
		return "application/octet-stream"
	}
}

// cryptoRandInt63 returns a cryptographically random non-negative int64.
func cryptoRandInt63() int64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("crypto/rand: " + err.Error())
	}
	return int64(binary.LittleEndian.Uint64(b[:]) & 0x7FFFFFFFFFFFFFFF)
}

// progressAdapter wraps our ProgressFunc as gotd's uploader.Progress.
type progressAdapter struct {
	fn ProgressFunc
}

func (p *progressAdapter) Chunk(_ context.Context, state uploader.ProgressState) error {
	if p.fn != nil {
		p.fn(state.Uploaded, state.Total)
	}
	return nil
}
