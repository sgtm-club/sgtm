package sgtmpb

import (
	"encoding/json"
	"fmt"
	"time"

	"moul.io/godev"
	"ultre.me/calcbiz/pkg/soundcloud"
)

func (p *Post) ApplyDefaults() {
	if p.Title == "" {
		p.Title = "noname"
	}
	if p.ArtworkURL == "" && p.Provider == Provider_SoundCloud {
		var metadata soundcloud.Track
		err := json.Unmarshal([]byte(p.ProviderMetadata), &metadata)
		if err == nil {
			fmt.Println(godev.PrettyJSON(metadata))
			p.ArtworkURL = metadata.User.AvatarURL
		}
	}
}

func (p *Post) CanonicalURL() string {
	return fmt.Sprintf("/post/%d", p.ID)
}

func (p *Post) GoDuration() time.Duration {
	return time.Millisecond * time.Duration(p.Duration)
}

func (u *User) CanonicalURL() string {
	return fmt.Sprintf("/@%s", u.Slug)
}
