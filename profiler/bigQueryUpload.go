// Copyright 2017 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package profiler is a client for the Cloud Profiler service.
//
// Usage example:
//
//	import "cloud.google.com/go/profiler"
//	...
//	if err := profiler.Start(profiler.Config{Service: "my-service"}); err != nil {
//	    // TODO: Handle error.
//	}
//
// This file consisting the util files required for uploading profiles to BigQuery

package profiler

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"log"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/google/uuid"
	_ "google.golang.org/protobuf/types/known/durationpb"
	_ "google.golang.org/protobuf/types/known/timestamppb"

	pprof_pb "github.com/google/pprof/profile"
)

type BigQueryProfileRow struct {
	ProfileUUID         string                 `bigquery:"profile_uuid"`
	UploadTimestamp     time.Time              `bigquery:"upload_timestamp"`
	ProfileType         string                 `bigquery:"profile_type"`
	TimeNanos           bigquery.NullTimestamp `bigquery:"time_nanos"`
	DurationNanos       int64                  `bigquery:"duration_nanos"`
	PeriodType          *BQValueType           `bigquery:"period_type"`
	Period              int64                  `bigquery:"period"`
	Comment             []string               `bigquery:"comment"` // NOTE: Populated as empty due to parsing limitations
	DefaultSampleType   string                 `bigquery:"default_sample_type"`
	DocURL              string                 `bigquery:"doc_url"` // NOTE: Populated as empty due to parsing limitations
	DropFramesPattern   string                 `bigquery:"drop_frames_pattern"`
	KeepFramesPattern   string                 `bigquery:"keep_frames_pattern"`
	SampleType          []BQValueType          `bigquery:"sample_type"`
	Sample              []BQSample             `bigquery:"sample"`
	Mapping             []BQMapping            `bigquery:"mapping"`
	Location            []BQLocation           `bigquery:"location"`
	Function            []BQFunction           `bigquery:"function"`
	DeploymentProjectID string                 `bigquery:"deployment_project_id"`
	DeploymentTarget    string                 `bigquery:"deployment_target"`
	DeploymentLabels    []BQKeyValue           `bigquery:"deployment_labels"`
	ProfileLabels       []BQKeyValue           `bigquery:"profile_labels"`
}
type BQValueType struct {
	Type string `bigquery:"type"`
	Unit string `bigquery:"unit"`
}
type BQSample struct {
	LocationID []int64   `bigquery:"location_id"`
	Value      []int64   `bigquery:"value"`
	Label      []BQLabel `bigquery:"label"`
}
type BQLabel struct {
	Key     string `bigquery:"key"`
	Str     string `bigquery:"str"`
	Num     int64  `bigquery:"num"`
	NumUnit string `bigquery:"num_unit"`
}
type BQMapping struct {
	ID              int64  `bigquery:"id"`
	MemoryStart     int64  `bigquery:"memory_start"`
	MemoryLimit     int64  `bigquery:"memory_limit"`
	FileOffset      int64  `bigquery:"file_offset"`
	Filename        string `bigquery:"filename"`
	BuildID         string `bigquery:"build_id"`
	HasFunctions    bool   `bigquery:"has_functions"`
	HasFilenames    bool   `bigquery:"has_filenames"`
	HasLineNumbers  bool   `bigquery:"has_line_numbers"`
	HasInlineFrames bool   `bigquery:"has_inline_frames"`
}
type BQLocation struct {
	ID        int64    `bigquery:"id"`
	MappingID int64    `bigquery:"mapping_id"`
	Address   int64    `bigquery:"address"`
	Line      []BQLine `bigquery:"line"`
	IsFolded  bool     `bigquery:"is_folded"`
}
type BQLine struct {
	FunctionID int64 `bigquery:"function_id"`
	Line       int64 `bigquery:"line"`
	Column     int64 `bigquery:"column"`
}
type BQFunction struct {
	ID         int64  `bigquery:"id"`
	Name       string `bigquery:"name"`
	SystemName string `bigquery:"system_name"`
	Filename   string `bigquery:"filename"`
	StartLine  int64  `bigquery:"start_line"`
}
type BQKeyValue struct {
	Key   string `bigquery:"key"`
	Value string `bigquery:"value"`
}

// parseAndUploadToBigQuery runs in a goroutine to handle BQ processing.
func (a *agent) parseAndUploadToBigQuery(ctx context.Context, profileTypeStr string, uploadTime time.Time, gzippedBytes []byte) {
	// 1. Decompress
	gzReader, err := gzip.NewReader(bytes.NewReader(gzippedBytes))
	if err != nil {
		debugLog("[BQ Upload] Failed to create gzip reader: %v. Skipping BQ upload.", err)
		return
	}
	decompressedBytes, err := io.ReadAll(gzReader)
	gzReader.Close()
	if err != nil {
		debugLog("[BQ Upload] Failed to decompress profile bytes: %v. Skipping BQ upload.", err)
		return
	}
	if len(decompressedBytes) == 0 {
		debugLog("[BQ Upload] Decompressed profile bytes are empty. Skipping BQ upload.")
		return
	}

	// 2. Unmarshal using pprof_pb.ParseData
	parsedProfile, err := pprof_pb.ParseData(decompressedBytes)
	if err != nil {
		debugLog("[BQ Upload] Failed to parse decompressed profile data: %v. Skipping BQ upload.", err)
		return
	}
	debugLog("[BQ Upload] Successfully parsed profile proto for type %s.", profileTypeStr)

	// 3. Denormalize into BigQueryProfileRow
	bqRow, err := a.denormalizeProfile(parsedProfile, profileTypeStr, uploadTime)
	if err != nil {
		debugLog("[BQ Upload] Failed during denormalization: %v. Skipping BQ upload.", err)
		return
	}

	// 4. Upload to BigQuery
	inserter := a.bqClient.DatasetInProject(a.deployment.ProjectId, a.bqDatasetID).Table(a.bqDenormTableID).Inserter()
	itemsToInsert := []*BigQueryProfileRow{bqRow}

	insertCtx, cancel := context.WithTimeout(context.Background(), config.BqUploadTimeout)
	defer cancel()

	if err := inserter.Put(insertCtx, itemsToInsert); err != nil {
		logBigQueryError(err, a.bqDenormTableID)
	} else {
		debugLog("[BQ Upload] Successfully uploaded denormalized profile %s to BigQuery table %s", bqRow.ProfileUUID, a.bqDenormTableID)
	}
}

// denormalizeProfile converts a parsed pprof_pb.Profile into a BigQueryProfileRow.
func (a *agent) denormalizeProfile(p *pprof_pb.Profile, profileTypeStr string, uploadTime time.Time) (*BigQueryProfileRow, error) {
	profileUUID := uuid.New().String()

	row := &BigQueryProfileRow{
		ProfileUUID:         profileUUID,
		UploadTimestamp:     uploadTime,
		ProfileType:         profileTypeStr,
		DurationNanos:       p.DurationNanos,
		Period:              p.Period,
		DeploymentProjectID: a.deployment.GetProjectId(),
		DeploymentTarget:    a.deployment.GetTarget(),
		DeploymentLabels:    mapToLabelPairs(a.deployment.GetLabels()),
		ProfileLabels:       mapToLabelPairs(a.profileLabels),

		// Access already resolved string fields from parsed profile
		// Comment field is not directly accessible/exported in parsed profile.Profile
		Comment:           []string{},          // Assign empty slice
		DefaultSampleType: p.DefaultSampleType, // Direct access (string)
		DropFramesPattern: p.DropFrames,        // Direct access (string)
		KeepFramesPattern: p.KeepFrames,        // Direct access (string)
		DocURL:            "",                  // No direct DocURL field
	}

	if p.TimeNanos != 0 {
		t := time.Unix(0, p.TimeNanos).UTC()
		row.TimeNanos = bigquery.NullTimestamp{Timestamp: t, Valid: true}
	} else {
		row.TimeNanos = bigquery.NullTimestamp{Valid: false}
	}

	if p.PeriodType != nil {
		row.PeriodType = &BQValueType{
			Type: p.PeriodType.Type,
			Unit: p.PeriodType.Unit,
		}
	}

	if len(p.SampleType) > 0 {
		row.SampleType = make([]BQValueType, len(p.SampleType))
		for i, st := range p.SampleType {
			row.SampleType[i] = BQValueType{
				Type: st.Type,
				Unit: st.Unit,
			}
		}
	} else {
		row.SampleType = []BQValueType{}
	}

	if len(p.Sample) > 0 {
		row.Sample = make([]BQSample, len(p.Sample))
		for i, s := range p.Sample {
			bqSample := BQSample{
				Value: s.Value,
			}
			// Location IDs
			if len(s.Location) > 0 {
				bqSample.LocationID = make([]int64, len(s.Location))
				for k, loc := range s.Location {
					if loc != nil {
						bqSample.LocationID[k] = int64(loc.ID)
					}
				}
			} else {
				bqSample.LocationID = []int64{}
			}
			if s.Value == nil {
				bqSample.Value = []int64{}
			}
			if len(s.Label) > 0 {
				bqSample.Label = make([]BQLabel, 0, len(s.Label))
				for key, values := range s.Label {
					if len(values) > 0 {
						bqSample.Label = append(bqSample.Label, BQLabel{
							Key: key,
							Str: values[0],
						})
					}
				}
			} else {
				bqSample.Label = []BQLabel{}
			}
			row.Sample[i] = bqSample
		}
	} else {
		row.Sample = []BQSample{}
	}

	if len(p.Mapping) > 0 {
		row.Mapping = make([]BQMapping, len(p.Mapping))
		for i, m := range p.Mapping {
			row.Mapping[i] = BQMapping{
				ID:              int64(m.ID),
				MemoryStart:     int64(m.Start),
				MemoryLimit:     int64(m.Limit),
				FileOffset:      int64(m.Offset),
				Filename:        m.File,
				BuildID:         m.BuildID,
				HasFunctions:    m.HasFunctions,
				HasFilenames:    m.HasFilenames,
				HasLineNumbers:  m.HasLineNumbers, // Correct field name
				HasInlineFrames: m.HasInlineFrames,
			}
		}
	} else {
		row.Mapping = []BQMapping{}
	}

	if len(p.Location) > 0 {
		row.Location = make([]BQLocation, len(p.Location))
		for i, l := range p.Location {
			bqLocation := BQLocation{
				ID:        int64(l.ID),
				MappingID: 0,
				Address:   int64(l.Address),
				IsFolded:  l.IsFolded,
			}
			if l.Mapping != nil {
				bqLocation.MappingID = int64(l.Mapping.ID)
			}
			// Line
			if len(l.Line) > 0 {
				bqLocation.Line = make([]BQLine, len(l.Line))
				for j, ln := range l.Line {
					bqLine := BQLine{
						FunctionID: 0,
						Line:       ln.Line,
						Column:     0,
					}
					if ln.Function != nil {
						bqLine.FunctionID = int64(ln.Function.ID)
					}
					bqLocation.Line[j] = bqLine
				}
			} else {
				bqLocation.Line = []BQLine{}
			}
			row.Location[i] = bqLocation
		}
	} else {
		row.Location = []BQLocation{}
	}

	if len(p.Function) > 0 {
		row.Function = make([]BQFunction, len(p.Function))
		for i, f := range p.Function {
			row.Function[i] = BQFunction{
				ID:         int64(f.ID),
				Name:       f.Name,
				SystemName: f.SystemName,
				Filename:   f.Filename,
				StartLine:  f.StartLine,
			}
		}
	} else {
		row.Function = []BQFunction{}
	}

	if row.DeploymentLabels == nil {
		row.DeploymentLabels = []BQKeyValue{}
	}
	if row.ProfileLabels == nil {
		row.ProfileLabels = []BQKeyValue{}
	}

	debugLog("[Denormalize] Denormalization complete for profile %s.", profileUUID)
	return row, nil
}

// logBigQueryError logs BQ insertion errors.
func logBigQueryError(err error, tableID string) {
	if multiErr, ok := err.(bigquery.PutMultiError); ok {
		for _, rowErr := range multiErr {
			debugLog("[BQ Upload] BigQuery row insertion error for table %s: index %d, errors: %v", tableID, rowErr.RowIndex, rowErr.Errors)
		}
		debugLog("[BQ Upload] Failed to insert some rows into BigQuery table %s.", tableID)
	} else {
		debugLog("[BQ Upload] Failed to upload profile to BigQuery table %s: %v", tableID, err)
	}
}

// mapToLabelPairs converts a map to BQ KeyValue slice.
func mapToLabelPairs(m map[string]string) []BQKeyValue {
	if len(m) == 0 {
		return []BQKeyValue{}
	}
	pairs := make([]BQKeyValue, 0, len(m))
	for k, v := range m {
		pairs = append(pairs, BQKeyValue{Key: k, Value: v})
	}
	return pairs
}

func closeBqClient(a *agent) {
	if a.bqClient != nil {
		debugLog("Closing BigQuery client...")
		if err := a.bqClient.Close(); err != nil {
			log.Printf("Cloud Profiler: Error closing BigQuery client: %v", err)
		} else {
			debugLog("BigQuery client closed.")
		}
	}
}
