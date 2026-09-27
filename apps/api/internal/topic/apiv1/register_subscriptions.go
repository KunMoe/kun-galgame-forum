package apiv1

import (
	"net/http"

	v1 "kun-galgame-api/internal/apiv1"

	"github.com/danielgtaylor/huma/v2"
)

func registerSubscriptions(api huma.API, x *Interactions) {
	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "getTopicSubscription",
		Method:      http.MethodGet,
		Path:        "/topics/{topic_id}/subscription",
		Summary:     "Get the caller's subscription to a topic",
		Description: "Returns how the caller hears about replies to this topic. A topic's author watches it from the moment it is posted; " +
			"everyone else starts at normal. " + interactionVisibility,
		Tags: []string{"topics"},
	}), x.getTopicSubscription)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "setTopicSubscription",
		Method:      http.MethodPut,
		Path:        "/topics/{topic_id}/subscription",
		Summary:     "Set the caller's subscription to a topic",
		Description: "Sets the caller's notification level for this topic. Setting the current level changes nothing. " +
			"Switching to watching starts the read position at the topic's last visible floor, so replies posted before it are not unread; " +
			"setting watching again while watching keeps the position. normal forgets the position. " + interactionVisibility,
		Tags: []string{"topics"},
	}), x.setTopicSubscription)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "markTopicRead",
		Method:      http.MethodPut,
		Path:        "/topics/{topic_id}/subscription/read-marker",
		Summary:     "Record how far the caller has read a topic",
		Description: "Moves a watching caller's read position forward to floor, capped at the topic's last visible floor; it never moves back. " +
			"When it reaches that floor, the caller's unread subscribed_topic_replied notification for this topic is marked read. " +
			"For normal and muted nothing is stored and the current state is returned. " + interactionVisibility,
		Tags: []string{"topics"},
	}), x.markTopicRead)

	huma.Register(api, v1.Required(huma.Operation{
		OperationID: "listTopicSubscriptions",
		Method:      http.MethodGet,
		Path:        "/me/topic-subscriptions",
		Summary:     "List the topics the caller watches",
		Description: "Lists the caller's watching subscriptions, the one where a reply landed most recently first, as a cursor page. " +
			"Each topic is judged again as getTopic would judge it for the caller; topics the caller can no longer read, topics by banned authors " +
			"and, unless include_nsfw, NSFW topics are left out and the server reads on to fill the page, so continue while next_cursor is present.",
		Tags: []string{"me"},
		Responses: problemResponses(map[int]string{
			400: "INVALID_PARAMETER, LIMIT_TOO_LARGE, or INVALID_CURSOR.",
			503: "SERVICE_UNAVAILABLE when the account service cannot render the topics' authors.",
		}),
	}), x.listTopicSubscriptions)
}
