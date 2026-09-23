package service

import (
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/pkg/userclient"
)

func appendUniqueStr(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}

func userBriefToDTO(u userclient.User) dto.UserBrief {
	return dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar}
}
