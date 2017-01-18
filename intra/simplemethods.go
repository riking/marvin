package intra

import (
	"context"
	"net/url"
	"strconv"
	"time"

	"github.com/patrickmn/go-cache"
)

var cacheLoginToUserID = cache.New(24*time.Hour, 4*time.Hour)

func (h *Helper) UserIDByLogin(ctx context.Context, login string) (int, error) {
	val, ok := cacheLoginToUserID.Get(login)
	if ok {
		return val.(int), nil
	}

	form := url.Values{}
	form.Set("filter[login]", login)

	var users []UserShort
	_, err := h.DoGetFormJSON(ctx, "/v2/users",
		form, &users)
	if err != nil {
		return -1, err
	}
	var id int = -1
	for _, v := range users {
		if v.Login == login {
			id = v.ID
			break
		}
	}
	cacheLoginToUserID.Add(login, id, 24*time.Hour)
	return id, nil
}

var cacheIDToUser = cache.New(24*time.Hour, 4*time.Hour)

func (h *Helper) UserByID(ctx context.Context, id int) (*User, error) {
	idStr := strconv.Itoa(id)
	val, ok := cacheIDToUser.Get(idStr)
	if ok {
		return val.(*User), nil
	}

	form := url.Values{}
	form.Set("id", idStr)
	var user *User
	_, err := h.DoGetFormJSON(ctx, "/v2/users/:id",
		form, &user)
	if err != nil {
		return nil, err
	}

	cacheIDToUser.Add(idStr, user, 24*time.Hour)
	return user, nil
}

var cacheIDToProject = cache.New(24*time.Hour, 4*time.Hour)
var cacheSlugToProject = cache.New(24*time.Hour, 4*time.Hour)

func (h *Helper) ProjectBySlug(ctx context.Context, slug string) (*Project, error) {
	val, ok := cacheSlugToProject.Get(slug)
	if ok {
		return val.(*Project), nil
	}

	form := url.Values{}
	form.Set("filter[slug]", slug)
	var projs []*Project
	_, err := h.DoGetFormJSON(ctx, "/v2/projects", form, &projs)
	if err != nil {
		return nil, err
	}
	var proj *Project
	for _, v := range projs {
		if v.Slug == slug {
			proj = v
			break
		}
	}
	cacheSlugToProject.Add(slug, proj, 24*time.Hour)
	cacheIDToProject.Add(strconv.Itoa(proj.ID), proj, 24*time.Hour)
	return proj, nil
}

func (h *Helper) ProjectByID(ctx context.Context, id int) (*Project, error) {
	idStr := strconv.Itoa(id)
	val, ok := cacheIDToProject.Get(idStr)
	if ok {
		return val.(*Project), nil
	}

	form := url.Values{}
	form.Set("filter[id]", idStr)
	var projs []*Project
	_, err := h.DoGetFormJSON(ctx, "/v2/projects", form, &projs)
	if err != nil {
		return nil, err
	}
	var proj *Project
	for _, v := range projs {
		if v.ID == id {
			proj = v
			break
		}
	}
	cacheSlugToProject.Add(proj.Slug, proj, 24*time.Hour)
	cacheIDToProject.Add(idStr, proj, 24*time.Hour)
	return proj, nil
}
