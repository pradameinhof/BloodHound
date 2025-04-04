// Copyright 2024 Specter Ops, Inc.
//
// Licensed under the Apache License, Version 2.0
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

package cypher

const (
	CountFunction              = "count"
	DateFunction               = "date"
	TimeFunction               = "time"
	LocalTimeFunction          = "localtime"
	DateTimeFunction           = "datetime"
	LocalDateTimeFunction      = "localdatetime"
	DurationFunction           = "duration"
	IdentityFunction           = "id"
	ToLowerFunction            = "tolower"
	ToUpperFunction            = "toupper"
	NodeLabelsFunction         = "labels"
	EdgeTypeFunction           = "type"
	StringSplitToArrayFunction = "split"
	ToStringFunction           = "tostring"
	ToIntegerFunction          = "toint"
	ListSizeFunction           = "size"
	CoalesceFunction           = "coalesce"
	CollectFunction            = "collect"

	// ITTC - Instant Type; Temporal Component (https://neo4j.com/docs/cypher-manual/current/functions/temporal/)
	ITTCYear              = "year"
	ITTCMonth             = "month"
	ITTCDay               = "day"
	ITTCHour              = "hour"
	ITTCMinute            = "minute"
	ITTCSecond            = "second"
	ITTCMillisecond       = "millisecond"
	ITTCMicrosecond       = "microsecond"
	ITTCNanosecond        = "nanosecond"
	ITTCTimeZone          = "timezone"
	ITTCEpochSeconds      = "epochseconds"
	ITTCEpochMilliseconds = "epochmillis"
)
