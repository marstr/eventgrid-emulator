package model

import (
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/eventgrid/eventgrid"
	egmgmt "github.com/Azure/azure-sdk-for-go/services/eventgrid/mgmt/2018-01-01/eventgrid"
)

var singletonSL *SubscriptionList

type SubscriptionList struct {
	subscribers map[string]egmgmt.EventSubscriptionFilter
}

func init() {
	singletonSL = NewSubscriptionList()
}

func NewSubscriptionList() *SubscriptionList {
	return &SubscriptionList{
		subscribers: make(map[string]egmgmt.EventSubscriptionFilter),
	}
}

func ListFilteredSubscribers(event eventgrid.Event) []string {
	return singletonSL.ListFilteredSubscribers(event)
}

// ListFilteredSubscribers applies the filter that was provided at the time the
// time each subscription was registered, and provides the handle
func (sl SubscriptionList) ListFilteredSubscribers(event eventgrid.Event) (results []string) {
	for value, filter := range sl.subscribers {
		if ApplyFilter(event, filter) {
			results = append(results, value)
		}
	}
	return
}

func Register(callback string, filter egmgmt.EventSubscriptionFilter) bool {
	return singletonSL.Register(callback, filter)
}

func (sl SubscriptionList) Register(callback string, filter egmgmt.EventSubscriptionFilter) (added bool) {
	_, added = sl.subscribers[callback]
	sl.subscribers[callback] = filter
	return
}

func Unregister(callback string) bool {
	return singletonSL.Unregister(callback)
}

func (sl SubscriptionList) Unregister(callback string) (removed bool) {
	if _, removed = sl.subscribers[callback]; removed {
		delete(sl.subscribers, callback)
	}
	return
}

// ApplyFilter determines whether or not an Event should advance past a filter.
// A return value of `true` implies the event should be processed, whereas `false` means
// the event does not match the specified criteria, and the subscriber should not be informed.
func ApplyFilter(event eventgrid.Event, filter egmgmt.EventSubscriptionFilter) bool {
	if !includesType(event, filter) {
		return false
	}

	var caseNormalizer func(string) string

	if event.Subject == nil {
		return false
	}

	if filter.SubjectBeginsWith == nil && filter.SubjectEndsWith == nil {
		return true
	}

	if filter.IsSubjectCaseSensitive != nil && *filter.IsSubjectCaseSensitive {
		caseNormalizer = func(a string) string {
			return a
		}
	} else {
		caseNormalizer = strings.ToUpper
	}

	matchesSubject := func(subject, substr string, substrFinder func(string, string) bool) bool {
		subject, substr = caseNormalizer(subject), caseNormalizer(substr)
		return substrFinder(subject, substr)
	}

	matchesPrefix := matchesSubject(*event.Subject, *filter.SubjectBeginsWith, strings.HasPrefix)
	matchesSuffix := matchesSubject(*event.Subject, *filter.SubjectEndsWith, strings.HasSuffix)
	return matchesPrefix || matchesSuffix
}

func includesType(event eventgrid.Event, filter egmgmt.EventSubscriptionFilter) (found bool) {
	if event.EventType == nil || filter.IncludedEventTypes == nil {
		return false
	}

	typeNameComparator := strings.EqualFold

	eventType := *event.EventType

	for _, includedEventType := range *filter.IncludedEventTypes {
		if typeNameComparator(includedEventType, eventType) || typeNameComparator(includedEventType, "all") {
			found = true
			break
		}
	}

	return found
}
