package intra

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
)

var cacheLoginToUserID = cache.New(24*time.Hour, 4*time.Hour)

func (h *Helper) UserIDByLogin(ctx context.Context, login string) (int, error) {
	login = strings.ToLower(login)
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
	slug = strings.ToLower(slug)
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

var cacheIDToCampus = cache.New(24*time.Hour, 4*time.Hour)
var cacheNameToCampus = cache.New(24*time.Hour, 4*time.Hour)

func (h *Helper) CampusByName(ctx context.Context, name string) (*Campus, error) {
	_, ok := cacheNameToCampus.Get("__DATALOADED")
	if !ok {
		err := h.loadAllCampus(ctx)
		if err != nil {
			return nil, err
		}
	}
	val, ok := cacheNameToCampus.Get(strings.ToLower(name))
	if ok {
		return val.(*Campus), nil
	} else {
		return nil, nil
	}
}

func (h *Helper) CampusByID(ctx context.Context, id int) (*Campus, error) {
	_, ok := cacheIDToCampus.Get("__DATALOADED")
	if !ok {
		err := h.loadAllCampus(ctx)
		if err != nil {
			return nil, err
		}
	}
	val, ok := cacheIDToCampus.Get(strconv.Itoa(id))
	if ok {
		return val.(*Campus), nil
	} else {
		return nil, nil
	}
}

func (h *Helper) loadAllCampus(ctx context.Context) error {
	form := url.Values{}
	var campuses []*Campus
	_, err := h.DoGetFormJSON(ctx, "/v2/campus", form, &campuses)
	if err != nil {
		return err
	}
	for _, v := range campuses {
		cacheNameToCampus.Add(strings.ToLower(v.Name), v, 24*time.Hour)
		cacheIDToCampus.Add(strconv.Itoa(v.ID), v, 24*time.Hour)
	}
	cacheNameToCampus.Add("__DATALOADED", nil, 23*time.Hour)
	cacheIDToCampus.Add("__DATALOADED", nil, 23*time.Hour)
	return nil
}
