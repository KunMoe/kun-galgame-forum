package service

import (
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/userclient"
)

func userBriefToDTO(u userclient.User) dto.UserBrief {
	return dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar}
}
