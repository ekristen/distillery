package forgejo

import "time"

type Release struct {
	ID          int64           `json:"id"`
	TagName     string          `json:"tag_name"`
	Name        string          `json:"name"`
	Body        string          `json:"body"`
	Draft       bool            `json:"draft"`
	Prerelease  bool            `json:"prerelease"`
	CreatedAt   time.Time       `json:"created_at"`
	PublishedAt time.Time       `json:"published_at"`
	Assets      []*ReleaseAsset `json:"assets"`
}

type ReleaseAsset struct {
	ID                 int64     `json:"id"`
	Name               string    `json:"name"`
	Size               int64     `json:"size"`
	DownloadCount      int64     `json:"download_count"`
	CreatedAt          time.Time `json:"created_at"`
	UUID               string    `json:"uuid"`
	BrowserDownloadURL string    `json:"browser_download_url"`
}
