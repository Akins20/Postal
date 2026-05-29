package twitter

import (
	"fmt"

	"github.com/Akins20/postal/internal/publish"
)

// Validate checks a post variant against X's constraints before any API call:
// weighted text length, media exclusivity (≤4 photos OR 1 GIF OR 1 video), and
// per-kind size limits. Returns a terminal publish.Error on violation.
func (a *Adapter) Validate(v publish.PostVariant) error {
	if err := a.validateText(v); err != nil {
		return err
	}
	return a.validateMedia(v.Media)
}

// validateText enforces the weighted 280-char limit (empty text is allowed only
// when media is attached).
func (a *Adapter) validateText(v publish.PostVariant) error {
	wl := weightedLength(v.Text)
	if wl > maxWeightedLen {
		return publish.Terminal("text_too_long",
			fmt.Sprintf("post is %d weighted characters; limit is %d", wl, maxWeightedLen), nil)
	}
	if wl == 0 && len(v.Media) == 0 {
		return publish.Terminal("empty_post", "post must have text or media", nil)
	}
	return nil
}

// mediaCounts tallies media by kind for the exclusivity/count checks.
type mediaCounts struct {
	images, gifs, videos int
}

// validateMedia enforces X's mutually-exclusive media rules and size caps.
func (a *Adapter) validateMedia(media []publish.MediaRef) error {
	if len(media) == 0 {
		return nil
	}
	counts, err := tallyMedia(media)
	if err != nil {
		return err
	}
	return checkMediaCounts(counts)
}

// tallyMedia counts media by kind and enforces per-item size limits.
func tallyMedia(media []publish.MediaRef) (mediaCounts, error) {
	var c mediaCounts
	for _, m := range media {
		switch m.Kind {
		case publish.MediaImage:
			c.images++
			if m.Bytes > maxImageBytes {
				return c, publish.Terminal("image_too_large", "image exceeds 5MB", nil)
			}
		case publish.MediaGIF:
			c.gifs++
			if m.Bytes > maxGIFBytes {
				return c, publish.Terminal("gif_too_large", "GIF exceeds 15MB", nil)
			}
		case publish.MediaVideo:
			c.videos++
			if m.Bytes > maxVideoBytes {
				return c, publish.Terminal("video_too_large", "video exceeds 512MB", nil)
			}
		default:
			return c, publish.Terminal("unsupported_media", "unsupported media kind: "+string(m.Kind), nil)
		}
	}
	return c, nil
}

// checkMediaCounts enforces per-kind maxima and the mutually-exclusive rule
// (up to 4 photos, OR 1 GIF, OR 1 video).
func checkMediaCounts(c mediaCounts) error {
	switch {
	case c.videos > maxVideos:
		return publish.Terminal("too_many_videos", "at most 1 video per post", nil)
	case c.gifs > maxGIFs:
		return publish.Terminal("too_many_gifs", "at most 1 GIF per post", nil)
	case c.images > maxImages:
		return publish.Terminal("too_many_images", "at most 4 images per post", nil)
	case (c.gifs > 0 || c.videos > 0) && (c.images > 0 || c.gifs+c.videos > 1):
		return publish.Terminal("mixed_media", "cannot mix media kinds; use up to 4 photos, or 1 GIF, or 1 video", nil)
	}
	return nil
}
