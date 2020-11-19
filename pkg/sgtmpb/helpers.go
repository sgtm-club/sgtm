package sgtmpb

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ultre.me/calcbiz/pkg/soundcloud"
)

// Post

func (p *Post) ApplyDefaults() {
	if p.ArtworkURL == "" && p.Provider == Provider_SoundCloud {
		var metadata soundcloud.Track
		err := json.Unmarshal([]byte(p.ProviderMetadata), &metadata)
		if err == nil {
			p.ArtworkURL = metadata.User.AvatarURL
		}
	}
}

func (p *Post) CanonicalURL() string {
	if p == nil {
		return "#"
	}
	return fmt.Sprintf("/post/%d", p.ID)
}

func (p *Post) GoDuration() time.Duration {
	return time.Millisecond * time.Duration(p.Duration)
}

func (p *Post) SafeDescription() string {
	if p.Body != "" {
		return p.Body
	}
	return p.ProviderDescription
}

func (p *Post) SafeTitle() string {
	if p.Title != "" {
		return p.Title
	}
	if p.ProviderTitle != "" {
		return p.ProviderTitle
	}
	return "noname"
}

func (p *Post) SafeLyrics() string {
	return strings.TrimSpace(p.Lyrics)
}

func (p *Post) Filter() {
	p.ProviderMetadata = ""
	p.DownloadURL = ""
}

func (p *Post) IsSoundCloud() bool { return p.GetProvider() == Provider_SoundCloud }
func (p *Post) IsIPFS() bool       { return p.GetProvider() == Provider_IPFS }

func (p *Post) IsSource() bool { return p.GetDawName() != "" }

func (p *Post) TagList() []string {
	if strings.TrimSpace(p.Tags) == "" {
		return nil
	}
	tags := strings.Split(p.Tags, ",")
	for idx, tag := range tags {
		tags[idx] = strings.TrimSpace(tag)
	}
	return tags
}

// User

func (u *User) ApplyDefaults() {

}

func (u *User) CanonicalURL() string {
	if u == nil {
		return "#"
	}
	return fmt.Sprintf("/@%s", u.Slug)
}

func (u *User) Fullname() string {
	fullname := fmt.Sprintf("%s %s", u.Firstname, u.Lastname)
	return strings.TrimSpace(fullname)
}

func (u *User) DisplayName() string {
	if u.Firstname != "" || u.Lastname != "" {
		return u.Fullname()
	}
	return fmt.Sprintf("@%s", u.Slug)
}

func (u *User) OtherLinksList() []string {
	links := strings.Split(strings.TrimSpace(u.OtherLinks), "\n")
	for idx, link := range links {
		links[idx] = strings.TrimSpace(link)
	}
	return links
}

func (u *User) HasSomethingAroundTheWeb() bool {
	return u.TwitterUsername != "" ||
		u.SoundcloudUsername != "" ||
		u.OtherLinks != "" ||
		u.Homepage != ""
}

func (u *User) Filter() {
	u.Email = ""
	u.DiscordUsername = ""
	u.DiscordID = ""
}
