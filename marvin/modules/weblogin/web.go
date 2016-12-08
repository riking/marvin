package weblogin

import "github.com/riking/homeapi/marvin/slack"

type WebContent struct {
	Title       string
	CurrentUser *slack.User

	NavbarCurrent     string
	NavbarItemsCustom interface{}

	BodyData interface{}
}

const (
	NavSectionFactoids = "Factoids"
	NavSectionInvite   = "Invite"
)

func (w *WebContent) NavbarItems() interface{} {
	if w.NavbarItemsCustom != nil {
		return w.NavbarItemsCustom
	}
	return []struct {
		Name string
		URL  string
	}{
		{Name: NavSectionFactoids, URL: "/factoids"},
		{Name: NavSectionInvite, URL: "/invite"},
	}
}
