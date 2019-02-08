// Code generated by protoc-gen-go. DO NOT EDIT.
// source: api/api.proto

package api

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import timestamp "github.com/golang/protobuf/ptypes/timestamp"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type AuthenticateFingerprint struct {
	Fingerprint          string   `protobuf:"bytes,1,opt,name=fingerprint,proto3" json:"fingerprint,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *AuthenticateFingerprint) Reset()         { *m = AuthenticateFingerprint{} }
func (m *AuthenticateFingerprint) String() string { return proto.CompactTextString(m) }
func (*AuthenticateFingerprint) ProtoMessage()    {}
func (*AuthenticateFingerprint) Descriptor() ([]byte, []int) {
	return fileDescriptor_api_f5fef02756c3ede7, []int{0}
}
func (m *AuthenticateFingerprint) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AuthenticateFingerprint.Unmarshal(m, b)
}
func (m *AuthenticateFingerprint) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AuthenticateFingerprint.Marshal(b, m, deterministic)
}
func (dst *AuthenticateFingerprint) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AuthenticateFingerprint.Merge(dst, src)
}
func (m *AuthenticateFingerprint) XXX_Size() int {
	return xxx_messageInfo_AuthenticateFingerprint.Size(m)
}
func (m *AuthenticateFingerprint) XXX_DiscardUnknown() {
	xxx_messageInfo_AuthenticateFingerprint.DiscardUnknown(m)
}

var xxx_messageInfo_AuthenticateFingerprint proto.InternalMessageInfo

func (m *AuthenticateFingerprint) GetFingerprint() string {
	if m != nil {
		return m.Fingerprint
	}
	return ""
}

type AuthenticateFacebook struct {
	Fingerprint          string   `protobuf:"bytes,1,opt,name=fingerprint,proto3" json:"fingerprint,omitempty"`
	Token                string   `protobuf:"bytes,2,opt,name=token,proto3" json:"token,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *AuthenticateFacebook) Reset()         { *m = AuthenticateFacebook{} }
func (m *AuthenticateFacebook) String() string { return proto.CompactTextString(m) }
func (*AuthenticateFacebook) ProtoMessage()    {}
func (*AuthenticateFacebook) Descriptor() ([]byte, []int) {
	return fileDescriptor_api_f5fef02756c3ede7, []int{1}
}
func (m *AuthenticateFacebook) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_AuthenticateFacebook.Unmarshal(m, b)
}
func (m *AuthenticateFacebook) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_AuthenticateFacebook.Marshal(b, m, deterministic)
}
func (dst *AuthenticateFacebook) XXX_Merge(src proto.Message) {
	xxx_messageInfo_AuthenticateFacebook.Merge(dst, src)
}
func (m *AuthenticateFacebook) XXX_Size() int {
	return xxx_messageInfo_AuthenticateFacebook.Size(m)
}
func (m *AuthenticateFacebook) XXX_DiscardUnknown() {
	xxx_messageInfo_AuthenticateFacebook.DiscardUnknown(m)
}

var xxx_messageInfo_AuthenticateFacebook proto.InternalMessageInfo

func (m *AuthenticateFacebook) GetFingerprint() string {
	if m != nil {
		return m.Fingerprint
	}
	return ""
}

func (m *AuthenticateFacebook) GetToken() string {
	if m != nil {
		return m.Token
	}
	return ""
}

type User struct {
	// The id of the user's account.
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	// The username of the user's account.
	Username string `protobuf:"bytes,2,opt,name=username,proto3" json:"username,omitempty"`
	// The display name of the user.
	DisplayName string `protobuf:"bytes,4,opt,name=display_name,json=displayName,proto3" json:"display_name,omitempty"`
	// A URL for an avatar image.
	AvatarUrl string `protobuf:"bytes,5,opt,name=avatar_url,json=avatarUrl,proto3" json:"avatar_url,omitempty"`
	// Additional information stored as a JSON object.
	Metadata string `protobuf:"bytes,6,opt,name=metadata,proto3" json:"metadata,omitempty"`
	// Indicates whether the user is currently online.
	Online bool `protobuf:"varint,8,opt,name=online,proto3" json:"online,omitempty"`
	// The UNIX time when the user was created.
	CreateTime *timestamp.Timestamp `protobuf:"bytes,9,opt,name=create_time,json=createTime,proto3" json:"create_time,omitempty"`
	// The UNIX time when the user was last updated.
	UpdateTime           *timestamp.Timestamp `protobuf:"bytes,10,opt,name=update_time,json=updateTime,proto3" json:"update_time,omitempty"`
	XXX_NoUnkeyedLiteral struct{}             `json:"-"`
	XXX_unrecognized     []byte               `json:"-"`
	XXX_sizecache        int32                `json:"-"`
}

func (m *User) Reset()         { *m = User{} }
func (m *User) String() string { return proto.CompactTextString(m) }
func (*User) ProtoMessage()    {}
func (*User) Descriptor() ([]byte, []int) {
	return fileDescriptor_api_f5fef02756c3ede7, []int{2}
}
func (m *User) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_User.Unmarshal(m, b)
}
func (m *User) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_User.Marshal(b, m, deterministic)
}
func (dst *User) XXX_Merge(src proto.Message) {
	xxx_messageInfo_User.Merge(dst, src)
}
func (m *User) XXX_Size() int {
	return xxx_messageInfo_User.Size(m)
}
func (m *User) XXX_DiscardUnknown() {
	xxx_messageInfo_User.DiscardUnknown(m)
}

var xxx_messageInfo_User proto.InternalMessageInfo

func (m *User) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *User) GetUsername() string {
	if m != nil {
		return m.Username
	}
	return ""
}

func (m *User) GetDisplayName() string {
	if m != nil {
		return m.DisplayName
	}
	return ""
}

func (m *User) GetAvatarUrl() string {
	if m != nil {
		return m.AvatarUrl
	}
	return ""
}

func (m *User) GetMetadata() string {
	if m != nil {
		return m.Metadata
	}
	return ""
}

func (m *User) GetOnline() bool {
	if m != nil {
		return m.Online
	}
	return false
}

func (m *User) GetCreateTime() *timestamp.Timestamp {
	if m != nil {
		return m.CreateTime
	}
	return nil
}

func (m *User) GetUpdateTime() *timestamp.Timestamp {
	if m != nil {
		return m.UpdateTime
	}
	return nil
}

type UserFriends struct {
	Friends              []*User  `protobuf:"bytes,1,rep,name=friends,proto3" json:"friends,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *UserFriends) Reset()         { *m = UserFriends{} }
func (m *UserFriends) String() string { return proto.CompactTextString(m) }
func (*UserFriends) ProtoMessage()    {}
func (*UserFriends) Descriptor() ([]byte, []int) {
	return fileDescriptor_api_f5fef02756c3ede7, []int{3}
}
func (m *UserFriends) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_UserFriends.Unmarshal(m, b)
}
func (m *UserFriends) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_UserFriends.Marshal(b, m, deterministic)
}
func (dst *UserFriends) XXX_Merge(src proto.Message) {
	xxx_messageInfo_UserFriends.Merge(dst, src)
}
func (m *UserFriends) XXX_Size() int {
	return xxx_messageInfo_UserFriends.Size(m)
}
func (m *UserFriends) XXX_DiscardUnknown() {
	xxx_messageInfo_UserFriends.DiscardUnknown(m)
}

var xxx_messageInfo_UserFriends proto.InternalMessageInfo

func (m *UserFriends) GetFriends() []*User {
	if m != nil {
		return m.Friends
	}
	return nil
}

type FriendRequest struct {
	UserId               string   `protobuf:"bytes,1,opt,name=user_id,json=userId,proto3" json:"user_id,omitempty"`
	Username             string   `protobuf:"bytes,2,opt,name=username,proto3" json:"username,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *FriendRequest) Reset()         { *m = FriendRequest{} }
func (m *FriendRequest) String() string { return proto.CompactTextString(m) }
func (*FriendRequest) ProtoMessage()    {}
func (*FriendRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_api_f5fef02756c3ede7, []int{4}
}
func (m *FriendRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_FriendRequest.Unmarshal(m, b)
}
func (m *FriendRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_FriendRequest.Marshal(b, m, deterministic)
}
func (dst *FriendRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_FriendRequest.Merge(dst, src)
}
func (m *FriendRequest) XXX_Size() int {
	return xxx_messageInfo_FriendRequest.Size(m)
}
func (m *FriendRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_FriendRequest.DiscardUnknown(m)
}

var xxx_messageInfo_FriendRequest proto.InternalMessageInfo

func (m *FriendRequest) GetUserId() string {
	if m != nil {
		return m.UserId
	}
	return ""
}

func (m *FriendRequest) GetUsername() string {
	if m != nil {
		return m.Username
	}
	return ""
}

type NotificationTokenUpdate struct {
	OldToken             string   `protobuf:"bytes,1,opt,name=old_token,json=oldToken,proto3" json:"old_token,omitempty"`
	Token                string   `protobuf:"bytes,2,opt,name=token,proto3" json:"token,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *NotificationTokenUpdate) Reset()         { *m = NotificationTokenUpdate{} }
func (m *NotificationTokenUpdate) String() string { return proto.CompactTextString(m) }
func (*NotificationTokenUpdate) ProtoMessage()    {}
func (*NotificationTokenUpdate) Descriptor() ([]byte, []int) {
	return fileDescriptor_api_f5fef02756c3ede7, []int{5}
}
func (m *NotificationTokenUpdate) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_NotificationTokenUpdate.Unmarshal(m, b)
}
func (m *NotificationTokenUpdate) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_NotificationTokenUpdate.Marshal(b, m, deterministic)
}
func (dst *NotificationTokenUpdate) XXX_Merge(src proto.Message) {
	xxx_messageInfo_NotificationTokenUpdate.Merge(dst, src)
}
func (m *NotificationTokenUpdate) XXX_Size() int {
	return xxx_messageInfo_NotificationTokenUpdate.Size(m)
}
func (m *NotificationTokenUpdate) XXX_DiscardUnknown() {
	xxx_messageInfo_NotificationTokenUpdate.DiscardUnknown(m)
}

var xxx_messageInfo_NotificationTokenUpdate proto.InternalMessageInfo

func (m *NotificationTokenUpdate) GetOldToken() string {
	if m != nil {
		return m.OldToken
	}
	return ""
}

func (m *NotificationTokenUpdate) GetToken() string {
	if m != nil {
		return m.Token
	}
	return ""
}

type UserUpdate struct {
	DisplayName          string   `protobuf:"bytes,1,opt,name=display_name,json=displayName,proto3" json:"display_name,omitempty"`
	Avatar               string   `protobuf:"bytes,2,opt,name=avatar,proto3" json:"avatar,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *UserUpdate) Reset()         { *m = UserUpdate{} }
func (m *UserUpdate) String() string { return proto.CompactTextString(m) }
func (*UserUpdate) ProtoMessage()    {}
func (*UserUpdate) Descriptor() ([]byte, []int) {
	return fileDescriptor_api_f5fef02756c3ede7, []int{6}
}
func (m *UserUpdate) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_UserUpdate.Unmarshal(m, b)
}
func (m *UserUpdate) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_UserUpdate.Marshal(b, m, deterministic)
}
func (dst *UserUpdate) XXX_Merge(src proto.Message) {
	xxx_messageInfo_UserUpdate.Merge(dst, src)
}
func (m *UserUpdate) XXX_Size() int {
	return xxx_messageInfo_UserUpdate.Size(m)
}
func (m *UserUpdate) XXX_DiscardUnknown() {
	xxx_messageInfo_UserUpdate.DiscardUnknown(m)
}

var xxx_messageInfo_UserUpdate proto.InternalMessageInfo

func (m *UserUpdate) GetDisplayName() string {
	if m != nil {
		return m.DisplayName
	}
	return ""
}

func (m *UserUpdate) GetAvatar() string {
	if m != nil {
		return m.Avatar
	}
	return ""
}

type Session struct {
	User                 *User    `protobuf:"bytes,1,opt,name=user,proto3" json:"user,omitempty"`
	Token                string   `protobuf:"bytes,2,opt,name=token,proto3" json:"token,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Session) Reset()         { *m = Session{} }
func (m *Session) String() string { return proto.CompactTextString(m) }
func (*Session) ProtoMessage()    {}
func (*Session) Descriptor() ([]byte, []int) {
	return fileDescriptor_api_f5fef02756c3ede7, []int{7}
}
func (m *Session) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Session.Unmarshal(m, b)
}
func (m *Session) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Session.Marshal(b, m, deterministic)
}
func (dst *Session) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Session.Merge(dst, src)
}
func (m *Session) XXX_Size() int {
	return xxx_messageInfo_Session.Size(m)
}
func (m *Session) XXX_DiscardUnknown() {
	xxx_messageInfo_Session.DiscardUnknown(m)
}

var xxx_messageInfo_Session proto.InternalMessageInfo

func (m *Session) GetUser() *User {
	if m != nil {
		return m.User
	}
	return nil
}

func (m *Session) GetToken() string {
	if m != nil {
		return m.Token
	}
	return ""
}

type Leaderboard struct {
	User                 *User    `protobuf:"bytes,1,opt,name=user,proto3" json:"user,omitempty"`
	Score                int64    `protobuf:"varint,2,opt,name=score,proto3" json:"score,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Leaderboard) Reset()         { *m = Leaderboard{} }
func (m *Leaderboard) String() string { return proto.CompactTextString(m) }
func (*Leaderboard) ProtoMessage()    {}
func (*Leaderboard) Descriptor() ([]byte, []int) {
	return fileDescriptor_api_f5fef02756c3ede7, []int{8}
}
func (m *Leaderboard) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Leaderboard.Unmarshal(m, b)
}
func (m *Leaderboard) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Leaderboard.Marshal(b, m, deterministic)
}
func (dst *Leaderboard) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Leaderboard.Merge(dst, src)
}
func (m *Leaderboard) XXX_Size() int {
	return xxx_messageInfo_Leaderboard.Size(m)
}
func (m *Leaderboard) XXX_DiscardUnknown() {
	xxx_messageInfo_Leaderboard.DiscardUnknown(m)
}

var xxx_messageInfo_Leaderboard proto.InternalMessageInfo

func (m *Leaderboard) GetUser() *User {
	if m != nil {
		return m.User
	}
	return nil
}

func (m *Leaderboard) GetScore() int64 {
	if m != nil {
		return m.Score
	}
	return 0
}

type LeaderboardRequest struct {
	Type                 string   `protobuf:"bytes,1,opt,name=type,proto3" json:"type,omitempty"`
	Mode                 string   `protobuf:"bytes,2,opt,name=mode,proto3" json:"mode,omitempty"`
	Page                 string   `protobuf:"bytes,3,opt,name=page,proto3" json:"page,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *LeaderboardRequest) Reset()         { *m = LeaderboardRequest{} }
func (m *LeaderboardRequest) String() string { return proto.CompactTextString(m) }
func (*LeaderboardRequest) ProtoMessage()    {}
func (*LeaderboardRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_api_f5fef02756c3ede7, []int{9}
}
func (m *LeaderboardRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_LeaderboardRequest.Unmarshal(m, b)
}
func (m *LeaderboardRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_LeaderboardRequest.Marshal(b, m, deterministic)
}
func (dst *LeaderboardRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_LeaderboardRequest.Merge(dst, src)
}
func (m *LeaderboardRequest) XXX_Size() int {
	return xxx_messageInfo_LeaderboardRequest.Size(m)
}
func (m *LeaderboardRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_LeaderboardRequest.DiscardUnknown(m)
}

var xxx_messageInfo_LeaderboardRequest proto.InternalMessageInfo

func (m *LeaderboardRequest) GetType() string {
	if m != nil {
		return m.Type
	}
	return ""
}

func (m *LeaderboardRequest) GetMode() string {
	if m != nil {
		return m.Mode
	}
	return ""
}

func (m *LeaderboardRequest) GetPage() string {
	if m != nil {
		return m.Page
	}
	return ""
}

type LeaderboardResponse struct {
	Items                []*Leaderboard `protobuf:"bytes,1,rep,name=items,proto3" json:"items,omitempty"`
	ItemCount            int32          `protobuf:"varint,2,opt,name=item_count,json=itemCount,proto3" json:"item_count,omitempty"`
	Page                 int32          `protobuf:"varint,3,opt,name=page,proto3" json:"page,omitempty"`
	HasNextPage          bool           `protobuf:"varint,4,opt,name=has_next_page,json=hasNextPage,proto3" json:"has_next_page,omitempty"`
	XXX_NoUnkeyedLiteral struct{}       `json:"-"`
	XXX_unrecognized     []byte         `json:"-"`
	XXX_sizecache        int32          `json:"-"`
}

func (m *LeaderboardResponse) Reset()         { *m = LeaderboardResponse{} }
func (m *LeaderboardResponse) String() string { return proto.CompactTextString(m) }
func (*LeaderboardResponse) ProtoMessage()    {}
func (*LeaderboardResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_api_f5fef02756c3ede7, []int{10}
}
func (m *LeaderboardResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_LeaderboardResponse.Unmarshal(m, b)
}
func (m *LeaderboardResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_LeaderboardResponse.Marshal(b, m, deterministic)
}
func (dst *LeaderboardResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_LeaderboardResponse.Merge(dst, src)
}
func (m *LeaderboardResponse) XXX_Size() int {
	return xxx_messageInfo_LeaderboardResponse.Size(m)
}
func (m *LeaderboardResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_LeaderboardResponse.DiscardUnknown(m)
}

var xxx_messageInfo_LeaderboardResponse proto.InternalMessageInfo

func (m *LeaderboardResponse) GetItems() []*Leaderboard {
	if m != nil {
		return m.Items
	}
	return nil
}

func (m *LeaderboardResponse) GetItemCount() int32 {
	if m != nil {
		return m.ItemCount
	}
	return 0
}

func (m *LeaderboardResponse) GetPage() int32 {
	if m != nil {
		return m.Page
	}
	return 0
}

func (m *LeaderboardResponse) GetHasNextPage() bool {
	if m != nil {
		return m.HasNextPage
	}
	return false
}

func init() {
	proto.RegisterType((*AuthenticateFingerprint)(nil), "spaceship.api.AuthenticateFingerprint")
	proto.RegisterType((*AuthenticateFacebook)(nil), "spaceship.api.AuthenticateFacebook")
	proto.RegisterType((*User)(nil), "spaceship.api.User")
	proto.RegisterType((*UserFriends)(nil), "spaceship.api.UserFriends")
	proto.RegisterType((*FriendRequest)(nil), "spaceship.api.FriendRequest")
	proto.RegisterType((*NotificationTokenUpdate)(nil), "spaceship.api.NotificationTokenUpdate")
	proto.RegisterType((*UserUpdate)(nil), "spaceship.api.UserUpdate")
	proto.RegisterType((*Session)(nil), "spaceship.api.Session")
	proto.RegisterType((*Leaderboard)(nil), "spaceship.api.Leaderboard")
	proto.RegisterType((*LeaderboardRequest)(nil), "spaceship.api.LeaderboardRequest")
	proto.RegisterType((*LeaderboardResponse)(nil), "spaceship.api.LeaderboardResponse")
}

func init() { proto.RegisterFile("api/api.proto", fileDescriptor_api_f5fef02756c3ede7) }

var fileDescriptor_api_f5fef02756c3ede7 = []byte{
	// 602 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x53, 0xcf, 0x6f, 0xd3, 0x30,
	0x14, 0x56, 0xfa, 0x6b, 0xed, 0xcb, 0xca, 0xc1, 0x9b, 0xb6, 0x50, 0x84, 0x28, 0xb9, 0xd0, 0x0b,
	0x19, 0x1a, 0xc7, 0x71, 0xd9, 0x40, 0x03, 0xa4, 0xa9, 0xaa, 0xd2, 0xed, 0xc2, 0x25, 0x72, 0x93,
	0xd7, 0xd6, 0x5a, 0x12, 0x1b, 0xdb, 0x41, 0xdb, 0x3f, 0xc3, 0x61, 0x47, 0xfe, 0x4a, 0x64, 0x3b,
	0x29, 0xeb, 0xd0, 0x98, 0xb8, 0xf9, 0xfb, 0xde, 0xf7, 0x3d, 0xbf, 0x7c, 0x2f, 0x86, 0x21, 0x15,
	0xec, 0x88, 0x0a, 0x16, 0x09, 0xc9, 0x35, 0x27, 0x43, 0x25, 0x68, 0x8a, 0x6a, 0xcd, 0x44, 0x44,
	0x05, 0x1b, 0xbd, 0x5a, 0x71, 0xbe, 0xca, 0xf1, 0xc8, 0x16, 0x17, 0xd5, 0xf2, 0x48, 0xb3, 0x02,
	0x95, 0xa6, 0x85, 0x70, 0xfa, 0xf0, 0x04, 0x0e, 0x4f, 0x2b, 0xbd, 0xc6, 0x52, 0xb3, 0x94, 0x6a,
	0x3c, 0x67, 0xe5, 0x0a, 0xa5, 0x90, 0xac, 0xd4, 0x64, 0x0c, 0xfe, 0xf2, 0x0f, 0x0c, 0xbc, 0xb1,
	0x37, 0x19, 0xc4, 0xf7, 0xa9, 0x70, 0x0a, 0xfb, 0x5b, 0x66, 0x9a, 0xe2, 0x82, 0xf3, 0xeb, 0xa7,
	0x9d, 0x64, 0x1f, 0xba, 0x9a, 0x5f, 0x63, 0x19, 0xb4, 0x6c, 0xcd, 0x81, 0xf0, 0xae, 0x05, 0x9d,
	0x2b, 0x85, 0x92, 0x3c, 0x83, 0x16, 0xcb, 0x6a, 0x5f, 0x8b, 0x65, 0x64, 0x04, 0xfd, 0x4a, 0xa1,
	0x2c, 0x69, 0x81, 0xb5, 0x63, 0x83, 0xc9, 0x6b, 0xd8, 0xcd, 0x98, 0x12, 0x39, 0xbd, 0x4d, 0x6c,
	0xbd, 0xe3, 0x6e, 0xab, 0xb9, 0xa9, 0x91, 0xbc, 0x04, 0xa0, 0x3f, 0xa8, 0xa6, 0x32, 0xa9, 0x64,
	0x1e, 0x74, 0xad, 0x60, 0xe0, 0x98, 0x2b, 0x99, 0x9b, 0xee, 0x05, 0x6a, 0x9a, 0x51, 0x4d, 0x83,
	0x9e, 0xeb, 0xde, 0x60, 0x72, 0x00, 0x3d, 0x5e, 0xe6, 0xac, 0xc4, 0xa0, 0x3f, 0xf6, 0x26, 0xfd,
	0xb8, 0x46, 0xe4, 0x04, 0xfc, 0x54, 0x22, 0xd5, 0x98, 0x98, 0x44, 0x83, 0xc1, 0xd8, 0x9b, 0xf8,
	0xc7, 0xa3, 0xc8, 0xc5, 0x1d, 0x35, 0x71, 0x47, 0x97, 0x4d, 0xdc, 0x31, 0x38, 0xb9, 0x21, 0x8c,
	0xb9, 0x12, 0xd9, 0xc6, 0x0c, 0x4f, 0x9b, 0x9d, 0xdc, 0x10, 0xe1, 0x07, 0xf0, 0x4d, 0x46, 0xe7,
	0x92, 0x61, 0x99, 0x29, 0xf2, 0x16, 0x76, 0x96, 0xee, 0x18, 0x78, 0xe3, 0xf6, 0xc4, 0x3f, 0xde,
	0x8b, 0xb6, 0x7e, 0x81, 0xc8, 0x88, 0xe3, 0x46, 0x13, 0x7e, 0x82, 0xa1, 0x73, 0xc6, 0xf8, 0xbd,
	0x42, 0xa5, 0xc9, 0x21, 0xec, 0x98, 0x28, 0x93, 0x4d, 0xde, 0x3d, 0x03, 0xbf, 0xfe, 0x33, 0xf3,
	0xf0, 0x02, 0x0e, 0xa7, 0x5c, 0xb3, 0xa5, 0x59, 0x3b, 0xe3, 0xe5, 0xa5, 0xd9, 0xde, 0x95, 0x1d,
	0x91, 0xbc, 0x80, 0x01, 0xcf, 0xb3, 0xc4, 0x6d, 0xd7, 0x75, 0xec, 0xf3, 0x3c, 0xb3, 0x92, 0x47,
	0xd6, 0xfe, 0x19, 0xc0, 0x0c, 0x59, 0x37, 0x78, 0xb8, 0x4f, 0xef, 0xef, 0x7d, 0x1e, 0x40, 0xcf,
	0x6d, 0xaf, 0xee, 0x53, 0xa3, 0xf0, 0x0b, 0xec, 0xcc, 0x51, 0x29, 0xc6, 0x4b, 0xf2, 0x06, 0x3a,
	0x66, 0x5a, 0xeb, 0x7e, 0x24, 0x13, 0x2b, 0x78, 0x64, 0xa4, 0x0b, 0xf0, 0x2f, 0x90, 0x66, 0x28,
	0x17, 0x9c, 0xca, 0xec, 0xbf, 0xba, 0xa9, 0x94, 0x4b, 0x97, 0x58, 0x3b, 0x76, 0x20, 0x9c, 0x01,
	0xb9, 0xd7, 0xad, 0x49, 0x9e, 0x40, 0x47, 0xdf, 0x8a, 0xe6, 0x03, 0xed, 0xd9, 0x70, 0x05, 0xcf,
	0x9a, 0xc0, 0xed, 0xd9, 0x70, 0x82, 0xae, 0x30, 0x68, 0x3b, 0xce, 0x9c, 0xc3, 0x9f, 0x1e, 0xec,
	0x6d, 0xb5, 0x54, 0x82, 0x97, 0x0a, 0xc9, 0x3b, 0xe8, 0x32, 0x8d, 0x45, 0xf3, 0x2f, 0x8c, 0x1e,
	0x4c, 0x7a, 0xdf, 0xe2, 0x84, 0xe6, 0x6d, 0x98, 0x43, 0x92, 0xf2, 0xaa, 0xd4, 0xf6, 0xde, 0x6e,
	0x3c, 0x30, 0xcc, 0x47, 0x43, 0x6c, 0x5d, 0xde, 0x75, 0x97, 0x93, 0x10, 0x86, 0x6b, 0xaa, 0x92,
	0x12, 0x6f, 0x74, 0x62, 0x8b, 0x1d, 0xfb, 0x34, 0xfc, 0x35, 0x55, 0x53, 0xbc, 0xd1, 0x33, 0xba,
	0xc2, 0xb3, 0x33, 0x78, 0xae, 0x65, 0x94, 0xf2, 0x22, 0xa2, 0x42, 0xa8, 0xed, 0x31, 0xce, 0x76,
	0xe7, 0x06, 0xce, 0xd7, 0x4c, 0x9c, 0x0a, 0x36, 0xf3, 0xbe, 0xb5, 0xa9, 0x60, 0x77, 0xad, 0xf6,
	0x7c, 0x3e, 0xfb, 0xd5, 0x1a, 0x6c, 0x6a, 0x8b, 0x9e, 0x7d, 0x09, 0xef, 0x7f, 0x07, 0x00, 0x00,
	0xff, 0xff, 0xac, 0xaa, 0xe4, 0x4f, 0xe3, 0x04, 0x00, 0x00,
}
