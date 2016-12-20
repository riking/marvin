package autoinvite

import "net/http"

func (mod *AutoInviteModule) registerHTTP() {
	mod.team.Router().HandleFunc("/invites", mod.HTTPListInvites)
	mod.team.Router().Path("/invites/{channel}").Methods(http.MethodPost).HandlerFunc(mod.HTTPInvite)
}

func (mod *AutoInviteModule) HTTPListInvites(w http.ResponseWriter, r *http.Request) {

}

func (mod *AutoInviteModule) HTTPInvite(w http.ResponseWriter, r *http.Request) {

}
