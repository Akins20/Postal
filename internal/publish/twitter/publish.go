package twitter

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/Akins20/postal/internal/channel"
	"github.com/Akins20/postal/internal/publish"
)

const (
	pathTweets      = "/2/tweets"
	pathMediaUpload = "/2/media/upload"

	// appendChunkBytes bounds each APPEND segment (X allows up to 5MB/chunk).
	appendChunkBytes = 4 << 20
	// maxStatusPolls bounds STATUS polling for async (video/GIF) processing.
	maxStatusPolls = 20
)

// createTweetRequest is the POST /2/tweets body.
type createTweetRequest struct {
	Text  string      `json:"text,omitempty"`
	Media *mediaField `json:"media,omitempty"`
	Reply *replyField `json:"reply,omitempty"`
}

type mediaField struct {
	MediaIDs []string `json:"media_ids"`
}

type replyField struct {
	InReplyToTweetID string `json:"in_reply_to_tweet_id"`
}

// Publish validates the variant, uploads any media, and creates the post.
func (a *Adapter) Publish(ctx context.Context, token channel.Token, v publish.PostVariant) (*publish.Result, error) {
	if err := a.Validate(v); err != nil {
		return nil, err
	}

	mediaIDs := make([]string, 0, len(v.Media))
	for i := range v.Media {
		id, err := a.uploadMedia(ctx, token.AccessToken, v.Media[i])
		if err != nil {
			return nil, err
		}
		mediaIDs = append(mediaIDs, id)
	}

	body := createTweetRequest{Text: v.Text}
	if len(mediaIDs) > 0 {
		body.Media = &mediaField{MediaIDs: mediaIDs}
	}
	if v.InReplyToID != "" {
		body.Reply = &replyField{InReplyToTweetID: v.InReplyToID}
	}

	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := a.postJSON(ctx, a.cfg.APIBaseURL+pathTweets, token.AccessToken, body, &resp); err != nil {
		return nil, err
	}
	if resp.Data.ID == "" {
		return nil, publish.Terminal("no_post_id", "platform returned no post id", nil)
	}
	return &publish.Result{PlatformPostID: resp.Data.ID}, nil
}

// uploadMedia runs the chunked INIT/APPEND/FINALIZE/STATUS sequence and returns
// the media_id to attach to a post.
func (a *Adapter) uploadMedia(ctx context.Context, bearer string, m publish.MediaRef) (string, error) {
	mediaID, err := a.mediaInit(ctx, bearer, m)
	if err != nil {
		return "", err
	}
	for seg, off := 0, 0; off < len(m.Data); seg, off = seg+1, off+appendChunkBytes {
		end := off + appendChunkBytes
		if end > len(m.Data) {
			end = len(m.Data)
		}
		if err := a.mediaAppend(ctx, bearer, mediaID, seg, m.Data[off:end]); err != nil {
			return "", err
		}
	}
	processing, err := a.mediaFinalize(ctx, bearer, mediaID)
	if err != nil {
		return "", err
	}
	if processing {
		if err := a.mediaAwait(ctx, bearer, mediaID); err != nil {
			return "", err
		}
	}
	return mediaID, nil
}

// mediaInit starts an upload and returns the media_id.
func (a *Adapter) mediaInit(ctx context.Context, bearer string, m publish.MediaRef) (string, error) {
	form := url.Values{}
	form.Set("command", "INIT")
	form.Set("total_bytes", strconv.Itoa(len(m.Data)))
	form.Set("media_type", m.MIME)
	form.Set("media_category", mediaCategory(m.Kind))

	var resp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := a.postForm(ctx, a.cfg.APIBaseURL+pathMediaUpload, bearer, form, &resp); err != nil {
		return "", err
	}
	if resp.Data.ID == "" {
		return "", publish.Terminal("media_init_failed", "media INIT returned no id", nil)
	}
	return resp.Data.ID, nil
}

// mediaAppend uploads one segment via multipart/form-data.
func (a *Adapter) mediaAppend(ctx context.Context, bearer, mediaID string, segment int, chunk []byte) error {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("command", "APPEND")
	_ = mw.WriteField("media_id", mediaID)
	_ = mw.WriteField("segment_index", strconv.Itoa(segment))
	part, err := mw.CreateFormFile("media", "chunk")
	if err != nil {
		return publish.Terminal("media_append_build", "could not build append request", err)
	}
	_, _ = part.Write(chunk)
	_ = mw.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.cfg.APIBaseURL+pathMediaUpload, &buf)
	if err != nil {
		return publish.Terminal("media_append_build", "could not build append request", err)
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return a.do(req, nil)
}

// mediaFinalize completes the upload, reporting whether async processing began.
func (a *Adapter) mediaFinalize(ctx context.Context, bearer, mediaID string) (bool, error) {
	form := url.Values{}
	form.Set("command", "FINALIZE")
	form.Set("media_id", mediaID)

	var resp struct {
		Data struct {
			ProcessingInfo *struct {
				State string `json:"state"`
			} `json:"processing_info"`
		} `json:"data"`
	}
	if err := a.postForm(ctx, a.cfg.APIBaseURL+pathMediaUpload, bearer, form, &resp); err != nil {
		return false, err
	}
	pi := resp.Data.ProcessingInfo
	return pi != nil && pi.State != "succeeded", nil
}

// mediaAwait polls STATUS until processing succeeds or fails.
func (a *Adapter) mediaAwait(ctx context.Context, bearer, mediaID string) error {
	for i := 0; i < maxStatusPolls; i++ {
		var resp struct {
			Data struct {
				ProcessingInfo *struct {
					State        string `json:"state"`
					CheckAfter   int    `json:"check_after_secs"`
					ErrorMessage string `json:"error_message"`
				} `json:"processing_info"`
			} `json:"data"`
		}
		url := fmt.Sprintf("%s%s?command=STATUS&media_id=%s", a.cfg.APIBaseURL, pathMediaUpload, mediaID)
		if err := a.getJSON(ctx, url, bearer, &resp); err != nil {
			return err
		}
		pi := resp.Data.ProcessingInfo
		switch {
		case pi == nil || pi.State == "succeeded":
			return nil
		case pi.State == "failed":
			return publish.Terminal("media_processing_failed", "media processing failed: "+pi.ErrorMessage, nil)
		}
		wait := time.Duration(pi.CheckAfter) * time.Second
		if wait <= 0 {
			wait = time.Second
		}
		select {
		case <-ctx.Done():
			return publish.Retryable("media_status_canceled", "context canceled during media processing", ctx.Err())
		case <-time.After(wait):
		}
	}
	return publish.Retryable("media_processing_timeout", "media processing did not complete in time", nil)
}

// postForm performs a Bearer-authenticated form-encoded POST decoding JSON out.
func (a *Adapter) postForm(ctx context.Context, url, bearer string, form url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return publish.Terminal("request_build_failed", "could not build request", err)
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return a.do(req, out)
}

// mediaCategory maps a media kind to X's media_category value.
func mediaCategory(k publish.MediaKind) string {
	switch k {
	case publish.MediaGIF:
		return "tweet_gif"
	case publish.MediaVideo:
		return "tweet_video"
	default:
		return "tweet_image"
	}
}
