package fc2

import (
	"encoding/json"

	"github.com/golang-jwt/jwt/v4"
)

type ControlToken struct {
	ChannelID      string `json:"channel_id,omitempty"`
	UserID         string `json:"user_id,omitempty"`
	Fc2ID          string `json:"fc2_id,omitempty"`
	OrzToken       string `json:"orz_token,omitempty"`
	SessionToken   string `json:"session_token,omitempty"`
	Premium        int    `json:"premium,omitempty"`
	Mode           string `json:"mode,omitempty"`
	Language       string `json:"language,omitempty"`
	ClientType     string `json:"client_type,omitempty"`
	ClientApp      string `json:"client_app,omitempty"`
	ClientVersion  string `json:"client_version,omitempty"`
	AppInstallKey  string `json:"app_install_key,omitempty"`
	ChannelVersion string `json:"channel_version,omitempty"`
	ControlTag     string `json:"control_tag,omitempty"`
	Ipv6           string `json:"ipv6,omitempty"`
	Commentable    bool   `json:"commentable,omitempty"`
	jwt.RegisteredClaims
}

type GetControlServerResponse struct {
	URL          string `json:"url"`
	Orz          string `json:"orz"`
	OrzRaw       string `json:"orz_raw"`
	ControlToken string `json:"control_token"`
	Status       int    `json:"status"`
}

type GetMetaResponse struct {
	Status int         `json:"status"`
	Data   GetMetaData `json:"data"`
}

type GetMetaData struct {
	ChannelData ChannelData `json:"channel_data"`
	ProfileData ProfileData `json:"profile_data"`
	UserData    UserData    `json:"user_data"`
}

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
	Fc2Channel          Fc2Channel            `json:"fc2_channel"`
	ControlTag          string                `json:"control_tag"`
	PublishMethod       string                `json:"publish_method"`
	VideoStereo3D       interface{}           `json:"video_stereo3d"`
	VideoMapping        interface{}           `json:"video_mapping"`
	VideoHorizontalView interface{}           `json:"video_horizontal_view"`
}

type Fc2Channel struct {
	Result      int           `json:"result"`
	UserID      int           `json:"userid"`
	Fc2ID       int           `json:"fc2id"`
	Adult       int           `json:"adult"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	URL         string        `json:"url"`
	Images      []interface{} `json:"images"`
}

type ChannelDataGiftList struct {
	ID   int      `json:"id"`
	Type int      `json:"type"`
	URL  []string `json:"url"`
	Name string   `json:"name"`
}

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

type UserData struct {
	IsLogin       int                `json:"is_login"`
	UserID        int                `json:"userid"`
	Fc2ID         int                `json:"fc2id"`
	Icon          string             `json:"icon"`
	Name          string             `json:"name"`
	Point         int                `json:"point"`
	AdultAccess   int                `json:"adult_access"`
	Recauth       int                `json:"recauth"`
	IsPremiumUser int                `json:"is_premium_user"`
	GiftList      []UserDataGiftList `json:"gift_list"`
	Stamina       Stamina            `json:"stamina"`
}

type UserDataGiftList struct {
	ID       int `json:"id"`
	Category int `json:"category"`
	Amount   int `json:"amount"`
}

type Stamina struct {
	Timestamp int             `json:"timestamp"`
	Stamina   [][]interface{} `json:"stamina"`
}

type WSResponse struct {
	ID        int             `json:"id,omitempty"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type CommentArguments struct {
	Comments []Comment `json:"comments"`
}

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

type HLSInformation struct {
	Status                 int        `json:"status"`
	Playlists              []Playlist `json:"playlists"`
	PlaylistsHighLatency   []Playlist `json:"playlists_high_latency"`
	PlaylistsMiddleLatency []Playlist `json:"playlists_middle_latency"`
}

type Playlist struct {
	Mode   int    `json:"mode"`
	Status int    `json:"status"`
	URL    string `json:"url"`
}

type ControlDisconnectionArguments struct {
	Code int `json:"code"`
}
