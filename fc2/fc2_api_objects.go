package fc2

import (
	"encoding/json"

	"github.com/golang-jwt/jwt/v5"
)

// ControlToken is the token used to authenticate with the FC2 API.
type ControlToken struct {
	ChannelID string `json:"channel_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	// Fc2ID is either a string when logged in, or the integer 0.
	Fc2ID          any `json:"fc2_id,omitempty"`
	OrzToken       any `json:"orz_token,omitempty"`
	SessionToken   any `json:"session_token,omitempty"`
	Premium        any `json:"premium,omitempty"`
	Mode           any `json:"mode,omitempty"`
	Language       any `json:"language,omitempty"`
	ClientType     any `json:"client_type,omitempty"`
	ClientApp      any `json:"client_app,omitempty"`
	ClientVersion  any `json:"client_version,omitempty"`
	AppInstallKey  any `json:"app_install_key,omitempty"`
	ChannelVersion any `json:"channel_version,omitempty"`
	ControlTag     any `json:"control_tag,omitempty"`
	Ipv6           any `json:"ipv6,omitempty"`
	Commentable    any `json:"commentable,omitempty"`
	ServiceID      any `json:"service_id,omitempty"`
	IP             any `json:"ip,omitempty"`
	UserName       any `json:"user_name,omitempty"`
	AdultAccess    any `json:"adult_access,omitempty"`
	AgentID        any `json:"agent_id,omitempty"`
	CountryCode    any `json:"country_code,omitempty"`
	PayMode        any `json:"pay_mode,omitempty"`
	jwt.RegisteredClaims
}

// GetControlServerResponse is the response from the get_control_server endpoint.
type GetControlServerResponse struct {
	URL          string `json:"url"`
	Orz          string `json:"orz"`
	OrzRaw       string `json:"orz_raw"`
	ControlToken string `json:"control_token"`
	Status       int    `json:"status"`
}

// GetMetaResponse is the response from the get_meta endpoint.
type GetMetaResponse struct {
	Status int         `json:"status"`
	Data   GetMetaData `json:"data"`
}

// GetMetaData is the data of the response from the get_meta endpoint.
type GetMetaData struct {
	ChannelData ChannelData `json:"channel_data"`
	ProfileData ProfileData `json:"profile_data"`
	UserData    UserData    `json:"user_data"`
}

// ChannelData describes the FC2 channel and stream.
type ChannelData struct {
	ChannelID           string                `json:"channelid"`
	UserID              string                `json:"userid"`
	Adult               int                   `json:"adult"`
	Twoshot             int                   `json:"twoshot"`
	Title               string                `json:"title"`
	Info                string                `json:"info"`
	Image               string                `json:"image"`
	LoginOnly           int                   `json:"login_only"`
	GiftLimit           int                   `json:"gift_limit"`
	GiftList            []ChannelDataGiftList `json:"gift_list"`
	CommentLimit        string                `json:"comment_limit"`
	Tfollow             int                   `json:"tfollow"`
	Tname               string                `json:"tname"`
	Fee                 int                   `json:"fee"`
	Amount              int                   `json:"amount"`
	Interval            int                   `json:"interval"`
	Category            string                `json:"category"`
	CategoryName        string                `json:"category_name"`
	IsOfficial          int                   `json:"is_official"`
	IsPremiumPublisher  int                   `json:"is_premium_publisher"`
	IsLinkShare         int                   `json:"is_link_share"`
	Ticketid            int                   `json:"ticketid"`
	IsPremium           int                   `json:"is_premium"`
	TicketPrice         int                   `json:"ticket_price"`
	TicketOnly          int                   `json:"ticket_only"`
	IsApp               int                   `json:"is_app"`
	IsVideo             int                   `json:"is_video"`
	IsREST              int                   `json:"is_rest"`
	Count               int                   `json:"count"`
	IsPublish           int                   `json:"is_publish"`
	IsLimited           int                   `json:"is_limited"`
	Start               int                   `json:"start"`
	Version             string                `json:"version"`
	FC2Channel          Channel               `json:"fc2_channel"`
	ControlTag          string                `json:"control_tag"`
	PublishMethod       string                `json:"publish_method"`
	VideoStereo3D       interface{}           `json:"video_stereo3d"`
	VideoMapping        interface{}           `json:"video_mapping"`
	VideoHorizontalView interface{}           `json:"video_horizontal_view"`
}

// Channel describes the FC2 channel.
type Channel struct {
	Result      int           `json:"result"`
	UserID      int           `json:"userid"`
	Fc2ID       int           `json:"fc2id"`
	Adult       int           `json:"adult"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	URL         string        `json:"url"`
	Images      []interface{} `json:"images"`
}

// ChannelDataGiftList describes the gifts that can be sent to the FC2 user.
type ChannelDataGiftList struct {
	ID   int      `json:"id"`
	Type int      `json:"type"`
	URL  []string `json:"url"`
	Name string   `json:"name"`
}

// ProfileData describes the FC2 user's profile.
type ProfileData struct {
	UserID string `json:"userid"`
	Fc2ID  string `json:"fc2id"`
	Name   string `json:"name"`
	Info   string `json:"info"`
	Icon   string `json:"icon"`
	Image  string `json:"image"`
	Sex    string `json:"sex"`
	Age    string `json:"age"`
}

// UserData describes the FC2 user.
type UserData struct {
	IsLogin       int         `json:"is_login"`
	UserID        int         `json:"userid"`
	Fc2ID         int         `json:"fc2id"`
	Icon          string      `json:"icon"`
	Name          string      `json:"name"`
	Point         interface{} `json:"point"`
	AdultAccess   interface{} `json:"adult_access"`
	Recauth       interface{} `json:"recauth"`
	IsPremiumUser interface{} `json:"is_premium_user"`
	GiftList      interface{} `json:"gift_list"`
	Stamina       interface{} `json:"stamina"`
}

// WSResponse is the response from the websocket.
type WSResponse struct {
	ID        int             `json:"id,omitempty"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// CommentArguments is the type of response corresponding to the "comment" event.
type CommentArguments struct {
	Comments []Comment `json:"comments"`
}

// Comment is the response from the websocket.
type Comment struct {
	UserName        string `json:"user_name"`
	Comment         string `json:"comment"`
	Timestamp       int    `json:"timestamp"`
	EncryptedUserID string `json:"encrypted_user_id"`
	OrzToken        string `json:"orz_token"`
	Hash            string `json:"hash"`
	Color           string `json:"color"`
	Size            string `json:"size"`
	Lang            string `json:"lang"`
	Anonymous       int    `json:"anonymous"`
	History         int    `json:"history"`
}

// HLSInformation is the response from the get_hls_information endpoint.
type HLSInformation struct {
	Status                 int        `json:"status"`
	Playlists              []Playlist `json:"playlists"`
	PlaylistsHighLatency   []Playlist `json:"playlists_high_latency"`
	PlaylistsMiddleLatency []Playlist `json:"playlists_middle_latency"`
}

// Playlist describes a m3u8 playlist and its specifications.
type Playlist struct {
	Mode   int    `json:"mode"`
	Status int    `json:"status"`
	URL    string `json:"url"`
}

// ControlDisconnectionArguments is the type of response corresponding to the "control_disconnection" event.
type ControlDisconnectionArguments struct {
	Code int `json:"code"`
}
