/*
 * This code was generated by
 * ___ _ _ _ _ _    _ ____    ____ ____ _    ____ ____ _  _ ____ ____ ____ ___ __   __
 *  |  | | | | |    | |  | __ |  | |__| | __ | __ |___ |\ | |___ |__/ |__|  | |  | |__/
 *  |  |_|_| | |___ | |__|    |__| |  | |    |__] |___ | \| |___ |  \ |  |  | |__| |  \
 *
 * Twilio - Insights
 * This is the public Twilio REST API.
 *
 * NOTE: This class is auto generated by OpenAPI Generator.
 * https://openapi-generator.tech
 * Do not edit the class manually.
 */

package openapi

import (
	"time"
)

// InsightsV1ConferenceParticipant struct for InsightsV1ConferenceParticipant
type InsightsV1ConferenceParticipant struct {
	// SID for this participant.
	ParticipantSid *string `json:"participant_sid,omitempty"`
	// The user-specified label of this participant.
	Label *string `json:"label,omitempty"`
	// The unique SID identifier of the Conference.
	ConferenceSid *string `json:"conference_sid,omitempty"`
	// Unique SID identifier of the call that generated the Participant resource.
	CallSid *string `json:"call_sid,omitempty"`
	// The unique SID identifier of the Account.
	AccountSid    *string `json:"account_sid,omitempty"`
	CallDirection *string `json:"call_direction,omitempty"`
	// Caller ID of the calling party.
	From *string `json:"from,omitempty"`
	// Called party.
	To         *string `json:"to,omitempty"`
	CallStatus *string `json:"call_status,omitempty"`
	// ISO alpha-2 country code of the participant based on caller ID or called number.
	CountryCode *string `json:"country_code,omitempty"`
	// Boolean. Indicates whether participant had startConferenceOnEnter=true or endConferenceOnExit=true.
	IsModerator *bool `json:"is_moderator,omitempty"`
	// ISO 8601 timestamp of participant join event.
	JoinTime *time.Time `json:"join_time,omitempty"`
	// ISO 8601 timestamp of participant leave event.
	LeaveTime *time.Time `json:"leave_time,omitempty"`
	// Participant durations in seconds.
	DurationSeconds *int `json:"duration_seconds,omitempty"`
	// Add Participant API only. Estimated time in queue at call creation.
	OutboundQueueLength *int `json:"outbound_queue_length,omitempty"`
	// Add Participant API only. Actual time in queue in seconds.
	OutboundTimeInQueue *int    `json:"outbound_time_in_queue,omitempty"`
	JitterBufferSize    *string `json:"jitter_buffer_size,omitempty"`
	// Boolean. Indicated whether participant was a coach.
	IsCoach *bool `json:"is_coach,omitempty"`
	// Call SIDs coached by this participant.
	CoachedParticipants *[]string `json:"coached_participants,omitempty"`
	ParticipantRegion   *string   `json:"participant_region,omitempty"`
	ConferenceRegion    *string   `json:"conference_region,omitempty"`
	CallType            *string   `json:"call_type,omitempty"`
	ProcessingState     *string   `json:"processing_state,omitempty"`
	// Participant properties and metadata.
	Properties *map[string]interface{} `json:"properties,omitempty"`
	// Object containing information of actions taken by participants. Contains a dictionary of URL links to nested resources of this Conference Participant.
	Events *map[string]interface{} `json:"events,omitempty"`
	// Object. Contains participant call quality metrics.
	Metrics *map[string]interface{} `json:"metrics,omitempty"`
	// The URL of this resource.
	Url *string `json:"url,omitempty"`
}
