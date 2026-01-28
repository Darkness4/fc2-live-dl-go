package api

import (
	"encoding/json"

	"github.com/golang-jwt/jwt/v5"
)

// ControlToken is the token used to authenticate with the FC2 API.
type ControlToken struct {
	ID                        string      `json:"id"`
	ChannelListChannelID      string      `json:"ChannelListchannel_id"`
	UserID                    any         `json:"user_id"` // TODO(HACK): either a Number or a String. The API is not consistent.
	ServiceID                 json.Number `json:"service_id"`
	OrzToken                  string      `json:"orz_token"`
	Premium                   json.Number `json:"premium"`
	Mode                      string      `json:"mode"`
	Language                  string      `json:"language"`
	ClientType                string      `json:"client_type"`
	ClientApp                 string      `json:"client_app"`
	ClientVersion             string      `json:"client_version"`
	AppInstallKey             string      `json:"app_install_key"`
	ChannelListChannelVersion string      `json:"ChannelListchannel_version"`
	IP                        string      `json:"ip"`
	Ipv6                      string      `json:"ipv6"`
	Commentable               json.Number `json:"commentable"`
	UserName                  string      `json:"user_name"`
	AdultAccess               json.Number `json:"adult_access"`
	AgentID                   json.Number `json:"agent_id"`
	CountryCode               string      `json:"country_code"`
	PayMode                   json.Number `json:"pay_mode"`
	Exp                       json.Number `json:"exp"`
	jwt.RegisteredClaims
}

// GetControlServerResponse is the response from the get_control_server endpoint.
type GetControlServerResponse struct {
	URL          string      `json:"url"`
	Orz          string      `json:"orz"`
	OrzRaw       string      `json:"orz_raw"`
	ControlToken string      `json:"control_token"`
	Status       json.Number `json:"status"`
}

// GetMetaResponse is the response from the get_meta endpoint.
type GetMetaResponse struct {
	Status json.Number `json:"status"`
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
	UserID              any                   `json:"userid"` // TODO(HACK): either a Number or a String. The API is not consistent.
	Adult               json.Number           `json:"adult"`
	Twoshot             json.Number           `json:"twoshot"`
	Title               string                `json:"title"`
	Info                string                `json:"info"`
	Image               string                `json:"image"`
	LoginOnly           json.Number           `json:"login_only"`
	GiftLimit           json.Number           `json:"gift_limit"`
	GiftList            []ChannelDataGiftList `json:"gift_list"`
	CommentLimit        any                   `json:"comment_limit"` // TODO(HACK): either a Number or a String. The API is not consistent.
	Tfollow             json.Number           `json:"tfollow"`
	Tname               string                `json:"tname"`
	Fee                 json.Number           `json:"fee"`
	Amount              json.Number           `json:"amount"`
	Interval            json.Number           `json:"interval"`
	Category            json.Number           `json:"category"`
	CategoryName        string                `json:"category_name"`
	IsOfficial          json.Number           `json:"is_official"`
	IsPremiumPublisher  json.Number           `json:"is_premium_publisher"`
	IsLinkShare         json.Number           `json:"is_link_share"`
	Ticketid            json.Number           `json:"ticketid"`
	IsPremium           json.Number           `json:"is_premium"`
	TicketPrice         json.Number           `json:"ticket_price"`
	TicketOnly          json.Number           `json:"ticket_only"`
	IsApp               json.Number           `json:"is_app"`
	IsVideo             json.Number           `json:"is_video"`
	IsREST              json.Number           `json:"is_rest"`
	Count               json.Number           `json:"count"`
	IsPublish           int64                 `json:"is_publish"`
	IsLimited           json.Number           `json:"is_limited"`
	Start               json.Number           `json:"start"`
	Version             string                `json:"version"`
	FC2Channel          Channel               `json:"fc2_channel"`
	ControlTag          string                `json:"control_tag"`
	PublishMethod       string                `json:"publish_method"`
	VideoStereo3D       any                   `json:"video_stereo3d"`
	VideoMapping        any                   `json:"video_mapping"`
	VideoHorizontalView any                   `json:"video_horizontal_view"`
}

// Channel describes the FC2 channel.
type Channel struct {
	Result      json.Number `json:"result"`
	UserID      any         `json:"userid"` // TODO(HACK): either a Number or a String. The API is not consistent.
	Fc2ID       any         `json:"fc2id"`  // TODO(HACK): either a Number or a String. The API is not consistent.
	Adult       json.Number `json:"adult"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	URL         string      `json:"url"`
	Images      []any       `json:"images"`
}

// ChannelDataGiftList describes the gifts that can be sent to the FC2 user.
type ChannelDataGiftList struct {
	ID   json.Number `json:"id"`
	Type json.Number `json:"type"`
	URL  []string    `json:"url"`
	Name string      `json:"name"`
}

// ProfileData describes the FC2 user's profile.
type ProfileData struct {
	UserID any         `json:"userid"` // TODO(HACK): either a Number or a String. The API is not consistent.
	Fc2ID  any         `json:"fc2id"`  // TODO(HACK): either a Number or a String. The API is not consistent.
	Name   string      `json:"name"`
	Info   string      `json:"info"`
	Icon   string      `json:"icon"`
	Image  string      `json:"image"`
	Sex    string      `json:"sex"`
	Age    json.Number `json:"age"`
}

// UserData describes the FC2 user.
type UserData struct {
	IsLogin       json.Number `json:"is_login"`
	UserID        any         `json:"userid"` // TODO(HACK): either a Number or a String. The API is not consistent.
	Fc2ID         any         `json:"fc2id"`  // TODO(HACK): either a Number or a String. The API is not consistent.
	Icon          string      `json:"icon"`
	Name          string      `json:"name"`
	Point         any         `json:"point"`
	AdultAccess   any         `json:"adult_access"`
	Recauth       any         `json:"recauth"`
	IsPremiumUser any         `json:"is_premium_user"`
	GiftList      any         `json:"gift_list"`
	Stamina       any         `json:"stamina"`
}

// WSResponse is the response from the websocket.
type WSResponse struct {
	ID        int64           `json:"id,omitempty"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

func (r WSResponse) String() string {
	b, _ := json.Marshal(r)
	return string(b)
}

// CommentArguments is the type of response corresponding to the "comment" event.
type CommentArguments struct {
	Comments []Comment `json:"comments"`
}

// Comment is the response from the websocket.
type Comment struct {
	UserName        string      `json:"user_name"`
	Comment         string      `json:"comment"`
	Timestamp       json.Number `json:"timestamp"`
	EncryptedUserID string      `json:"encrypted_user_id"`
	OrzToken        string      `json:"orz_token"`
	Hash            string      `json:"hash"`
	Color           string      `json:"color"`
	Size            string      `json:"size"`
	Lang            string      `json:"lang"`
	Anonymous       json.Number `json:"anonymous"`
	History         json.Number `json:"history"`
}

// HLSInformation is the response from the get_hls_information endpoint.
type HLSInformation struct {
	Status                 json.Number `json:"status"`
	Playlists              []Playlist  `json:"playlists"`
	PlaylistsHighLatency   []Playlist  `json:"playlists_high_latency"`
	PlaylistsMiddleLatency []Playlist  `json:"playlists_middle_latency"`
}

// Playlist describes a m3u8 playlist and its specifications.
type Playlist struct {
	Mode   int         `json:"mode"`
	Status json.Number `json:"status"`
	URL    string      `json:"url"`
}

// ControlDisconnectionArguments is the type of response corresponding to the "control_disconnection" event.
type ControlDisconnectionArguments struct {
	Code int `json:"code"`
}

// GetChannelListResponse is the response from the get_channel_list endpoint.
type GetChannelListResponse struct {
	Link    string                  `json:"link"`
	IsAdult int64                   `json:"is_adult"`
	Time    int64                   `json:"time"`
	Channel []GetChannelListChannel `json:"channel"`
}

// GetChannelListChannel describes the FC2 channel.
type GetChannelListChannel struct {
	ID             string      `json:"id"`
	Bid            string      `json:"bid"`
	Video          json.Number `json:"video"`
	App            json.Number `json:"app"`
	Category       json.Number `json:"category"`
	Type           json.Number `json:"type"`
	Fc2ID          any         `json:"fc2id"` // TODO(HACK): either a Number or a String. The API is not consistent.
	Name           string      `json:"name"`
	Title          string      `json:"title"`
	Image          string      `json:"image"`
	Start          string      `json:"start"`
	StartTime      json.Number `json:"start_time"`
	Sex            string      `json:"sex"`
	Pay            json.Number `json:"pay"`
	Interval       json.Number `json:"interval"`
	Amount         json.Number `json:"amount"`
	Lang           string      `json:"lang"`
	Total          json.Number `json:"total"`
	Count          json.Number `json:"count"`
	Login          json.Number `json:"login"`
	CommentL       json.Number `json:"comment_l"`
	Tid            json.Number `json:"tid"`
	Price          json.Number `json:"price"`
	Official       json.Number `json:"official"`
	CommentScore   json.Number `json:"comment_score"`
	DenyCountryFlg string      `json:"deny_country_flg"`
	Panorama       json.Number `json:"panorama"`
}
