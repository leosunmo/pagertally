package process

import (
	"fmt"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/leosunmo/pagertally/pkg/datasources"
	"github.com/leosunmo/pagertally/pkg/timespan"
)

// Intersector returns attributed spans of provided spans that intersect with any of the attribution datasources
type Intersector func([]timespan.Span) []timespan.AttributedSpan

// businessHourIntersector is a simple intersector that fills in the non-attributed time with business hour attribute
func businessHoursIntersector(spans []timespan.Span) []timespan.AttributedSpan {
	out := make([]timespan.AttributedSpan, len(spans))
	for i := range spans {
		out[i] = timespan.AttributedSpan{
			Span:     spans[i],
			SpanType: timespan.Business,
		}
	}
	return out
}

// genIntersectorFromDatasourceSpans returns an Intersector from the provided on call attribute and the spans associated with it
func genIntersectorFromDatasourceSpans(attr timespan.OnCallAttribute, matchSpans []timespan.Span) Intersector {
	return func(testSpans []timespan.Span) []timespan.AttributedSpan {
		// Write code that finds any spans inside testSpans that are also inside matchSpans
		out := []timespan.AttributedSpan{}
		for _, testSpan := range testSpans {
			for _, matchSpan := range matchSpans {
				matchedPortionSpan, overlap := testSpan.Intersection(matchSpan)
				if overlap {
					out = append(out, timespan.AttributedSpan{
						Span:     matchedPortionSpan,
						SpanType: attr,
					})
				}
			}
		}
		return out
	}
}

// ScheduleUserShifts processes all user shifts for all Pagerduty schedules and
// returns a slice of attributed user shifts with the user and PD schedule as values of that struct
func ScheduleUserShifts(schedUserShifts timespan.ScheduleUserShifts, companyDayDatasource, calendarDatasource, weekendDatasource, afterHoursDatasource datasources.DataSource) map[string][]timespan.UserShiftResults {
	output := map[string][]timespan.UserShiftResults{}
	for schedule, userShifts := range schedUserShifts {
		userResults := []timespan.UserShiftResults{}
		// DEBUG
		var totalDurs time.Duration
		// DEBUG
		for user, shifts := range userShifts {
			attrShifts := attributeShift(shifts, companyDayDatasource, calendarDatasource, weekendDatasource, afterHoursDatasource)
			singleResult := timespan.UserShiftResults{
				Schedule:  schedule,
				User:      user,
				Shifts:    shifts,
				Breakdown: attrShifts,
			}
			// DEBUG
			if log.GetLevel() == log.TraceLevel {
				fmt.Printf("%s's shifts:\n", user.Name)
				var shiftsDuration time.Duration
				for i, shift := range shifts {
					shiftsDuration = shiftsDuration + shift.End().Sub(shift.Start())
					fmt.Printf("\tShift %d:\n\t%s  -  %s\n\tDuration: %s\n\n", i, shift.Start(), shift.End(), shift.End().Sub(shift.Start()))
				}
				fmt.Printf("\nTotal shift duration: %s\n", shiftsDuration)
				fmt.Printf("\n%s's breakdown:\n", user.Name)

				var attrShiftsDuration time.Duration
				for i, shift := range attrShifts {
					attrShiftsDuration = attrShiftsDuration + shift.End().Sub(shift.Start())
					fmt.Printf("\tAttribShift %d Type: %d:\n\t%s  -  %s\n\tDuration: %s\n\n", i, shift.SpanType, shift.Start(), shift.End(), shift.End().Sub(shift.Start()))
				}
				fmt.Printf("Total attribShift duration: %s\n", attrShiftsDuration)
				fmt.Println()
				if shiftsDuration-attrShiftsDuration != 0 {
					fmt.Println("===============================================================")
					fmt.Printf("Shifts and attrShifts mismatch, %s vs %s\n", shiftsDuration, attrShiftsDuration)
					fmt.Println("===============================================================")
				}
				totalDurs = totalDurs + attrShiftsDuration
			}
			// END TRACE
			userResults = append(userResults, singleResult)
		}
		// DEBUG
		if log.GetLevel() == log.DebugLevel {
			log.Debugf("Total time from schedule %s : %s", schedule, totalDurs)
		}
		// END DEBUG
		output[string(schedule)] = userResults
	}
	return output
}

// attributeShift returns timespans with added oncall attributes for the whole shift
func attributeShift(spans []timespan.Span, companyDayDatasource, calendarDatasource, weekendDatasource, afterHoursDatasource datasources.DataSource) []timespan.AttributedSpan {

	var deciders = []Intersector{
		genIntersectorFromDatasourceSpans(timespan.CompanyDay, companyDayDatasource.Spans()),
		genIntersectorFromDatasourceSpans(timespan.StatHoliday, calendarDatasource.Spans()),
		genIntersectorFromDatasourceSpans(timespan.Weekend, weekendDatasource.Spans()),
		genIntersectorFromDatasourceSpans(timespan.AfterHours, afterHoursDatasource.Spans()),
		businessHoursIntersector,
	}
	output := timespan.AttributedSpans{}
	for _, decider := range deciders {
		matches := decider(spans)
		output = append(output, matches...)
		spans = removeMatchedSpans(spans, matches)
	}
	sort.Sort(output)
	return output
}

// removeMatchedSpans returns all spans that were not present in the matches slice.
// if the span matches partially, we return the span with the matched part removed
func removeMatchedSpans(spans []timespan.Span, matches []timespan.AttributedSpan) []timespan.Span {
	reprocess := []timespan.Span{}
	if len(matches) < 1 {
		return spans
	}
	for _, span := range spans {
		leftovers := compareSpanToMatches(span, matches)
		reprocess = append(reprocess, leftovers...)
	}
	output := timespan.Deduplicate(reprocess)
	return output
}

// matchAndReturnLeftovers matches match with testspan and if it ovelaps returns the leftover spans.
// If it overlaps perfectly, return true with empty span
// If there is no overlap at all, returns false with empty span
func matchAndReturnLeftovers(match, testSpan timespan.Span) ([]timespan.Span, bool) {
	var leftovers = []timespan.Span{}
	var overlappingSpan = timespan.Span{}
	var overlap = false
	if overlappingSpan, overlap = testSpan.Intersection(match); !overlap {
		// There was no overlap between the shift span and this match.
		return []timespan.Span{}, false
	} // There is overlap between the shiftspan and the matched span

	if testSpan.Start().Before(overlappingSpan.Start()) { // Check if there's any shiftspan left over *before* the overlap
		leftovers = append(leftovers, timespan.New(testSpan.Start(), overlappingSpan.Start()))
	}

	if testSpan.End().After(overlappingSpan.End()) { // Check if there's any shiftspan left over *after* the overlap
		spanStartTime := overlappingSpan.End()
		if timespan.IsEndOfDay(overlappingSpan.End()) {
			// If leftovers are about to start with 23:59:59 we move it forward to next day
			spanStartTime = timespan.StartOfDay(overlappingSpan.End().AddDate(0, 0, 1))
		}
		leftovers = append(leftovers, timespan.New(spanStartTime, testSpan.End()))
	}

	return leftovers, true
}

func compareSpanToMatches(testSpan timespan.Span, matches []timespan.AttributedSpan) []timespan.Span {
	leftovers := []timespan.Span{}
	var overallMatch = false
	for _, match := range matches {
		newLeftovers, matched := matchAndReturnLeftovers(match.Span, testSpan)
		if !matched {
			continue
		}
		if len(newLeftovers) == 0 {
			return leftovers // Move to next span as this one is completely matched
		}
		// There was a match and there are leftovers
		overallMatch = true

		// Check if the new leftovers match with any other match by going through them again
		secondStageLeftovers := []timespan.Span{}
		secondStageCompleteMatch := false
		for _, secondMatch := range matches {
			if match.Span.Equal(secondMatch.Span) { // We're already working with this match
				continue
			}
			for _, newLeftover := range newLeftovers {
				if newLeftoverMatches, overlap := matchAndReturnLeftovers(secondMatch.Span, newLeftover); overlap {
					if len(newLeftoverMatches) == 0 {
						if len(newLeftovers) == 1 {
							return leftovers
						}
						secondStageCompleteMatch = true
						continue // This leftover is a complete match with another Match, let's check the other leftovers
					}
					secondStageCompleteMatch = false
					secondStageLeftovers = append(secondStageLeftovers, newLeftoverMatches...) // There were some subleftovers, add them for reprocessing
				}
			}
		}
		if len(secondStageLeftovers) != 0 {
			newLeftovers = secondStageLeftovers
		}
		thirdStageCompleteMatch := false
		thirdStageLeftovers := []timespan.Span{}
		for _, leftover := range leftovers {
			for _, newLeftover := range newLeftovers {
				// Check if this new leftover piece matches an older leftover piece
				if scrapLeftover, overlap := matchAndReturnLeftovers(newLeftover, leftover); overlap {
					if len(scrapLeftover) == 0 {
						// Check if it's a perfect match. If so, just mark it as processed and move on
						if len(newLeftovers) == 1 {
							return leftovers
						}
						thirdStageCompleteMatch = true
						continue
					}
					// We found a leftover that overlaps with our new leftover, but it's not a perfect match
					thirdStageLeftovers = append(thirdStageLeftovers, scrapLeftover...)
					thirdStageCompleteMatch = false
				}
			}
		}
		if secondStageCompleteMatch && thirdStageCompleteMatch {
			return leftovers
		}
		if len(thirdStageLeftovers) != 0 {
			newLeftovers = thirdStageLeftovers
		}
		leftovers = append(leftovers, newLeftovers...)

	}
	if !overallMatch {
		// No matches overlapped with the span, let's add it for reprocessing
		leftovers = append(leftovers, testSpan)
	}
	dedupLeftovers := timespan.Deduplicate(leftovers)
	return dedupLeftovers
}
