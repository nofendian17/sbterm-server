package domain

// NotificationList is one page of notifications plus the unread count.
type NotificationList struct {
	Unread int
	Items  []Notification
}

// Notification is one entry in the user's notification feed.
type Notification struct {
	ID      int64
	Type    string
	Avatar  string
	Message string
	Masks   []NotificationMask
	LinkTo  NotificationLink
	Created string
	IsRead  bool
}

// NotificationMask resolves one %key% placeholder in Message.
type NotificationMask struct {
	Key  string
	Text string
	Tag  string
	Type string
}

// NotificationLink is the navigation target of the notification.
type NotificationLink struct {
	Key   int64
	Value string
}
