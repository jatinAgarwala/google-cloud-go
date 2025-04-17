// package main

// import (
// 	"bytes"
// 	"compress/gzip"
// 	"context" // Added for BigQuery client
// 	"encoding/base64"

// 	// "encoding/json" // No longer needed for printing JSON
// 	"errors"
// 	"fmt"
// 	"io"
// 	"log"
// 	"time"

// 	profilepb "profilerbqtest/proto" // Your generated proto package

// 	"cloud.google.com/go/bigquery" // Added BigQuery client
// 	"github.com/google/uuid"       // Added for UUID generation
// 	"google.golang.org/protobuf/proto"
// )

// // --- BigQuery Table Details ---
// const (
// 	projectID = "dogwood-harmony-449407-q2"
// 	datasetID = "jatinagarwala"
// 	tableID   = "profiler_denormalised_data"
// )

// // --- Sample Profile Data ---
// // Replace this with your actual base64 encoded gzipped profile bytes
// // For testing upload logic, even the placeholder might work if parsing doesn't fail,
// // but using real data is better.
// // var gzippedProfileBytesBase64 = `H4sIAAAAAAAA/ytJLS4BAAxGw7gEAAAA` // <<<=== REPLACE WITH YOUR ACTUAL BASE64 STRING (or leave placeholder for basic test)
// var gzippedProfileBytesBase64 = `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` // <<<=== REPLACE WITH YOUR ACTUAL BASE64 STRING (or leave placeholder for basic test)

// // --- BigQuery Schema Mirror Structs (with bigquery tags) ---
// type BigQueryProfileRow struct {
// 	ProfileUUID     string    `bigquery:"profile_uuid"`
// 	UploadTimestamp time.Time `bigquery:"upload_timestamp"` // Required, so non-pointer time.Time is fine
// 	ProfileType     string    `bigquery:"profile_type"`
// 	// Change TimeNanos to use bigquery.NullTimestamp
// 	TimeNanos           bigquery.NullTimestamp `bigquery:"time_nanos"`
// 	DurationNanos       int64                  `bigquery:"duration_nanos"`
// 	PeriodType          *BQValueType           `bigquery:"period_type"`
// 	Period              int64                  `bigquery:"period"`
// 	Comment             []string               `bigquery:"comment"`
// 	DefaultSampleType   string                 `bigquery:"default_sample_type"`
// 	DocURL              string                 `bigquery:"doc_url"`
// 	DropFramesPattern   string                 `bigquery:"drop_frames_pattern"`
// 	KeepFramesPattern   string                 `bigquery:"keep_frames_pattern"`
// 	SampleType          []BQValueType          `bigquery:"sample_type"`
// 	Sample              []BQSample             `bigquery:"sample"`
// 	Mapping             []BQMapping            `bigquery:"mapping"`
// 	Location            []BQLocation           `bigquery:"location"`
// 	Function            []BQFunction           `bigquery:"function"`
// 	DeploymentProjectID string                 `bigquery:"deployment_project_id"`
// 	DeploymentTarget    string                 `bigquery:"deployment_target"`
// 	DeploymentLabels    []BQKeyValue           `bigquery:"deployment_labels"`
// 	ProfileLabels       []BQKeyValue           `bigquery:"profile_labels"`
// }

// // BQValueType represents the nested structure for value types (sample_type, period_type)
// type BQValueType struct {
// 	Type string `bigquery:"type"`
// 	Unit string `bigquery:"unit"`
// }

// // BQSample represents the nested structure for samples
// type BQSample struct {
// 	LocationID []int64   `bigquery:"location_id"` // Changed to int64
// 	Value      []int64   `bigquery:"value"`
// 	Label      []BQLabel `bigquery:"label"`
// }

// // BQLabel represents the nested structure for sample labels
// type BQLabel struct {
// 	Key     string `bigquery:"key"`
// 	Str     string `bigquery:"str"`
// 	Num     int64  `bigquery:"num"`
// 	NumUnit string `bigquery:"num_unit"`
// }

// // BQMapping represents the nested structure for mappings
// type BQMapping struct {
// 	ID              int64  `bigquery:"id"`           // Changed to int64
// 	MemoryStart     int64  `bigquery:"memory_start"` // Changed to int64
// 	MemoryLimit     int64  `bigquery:"memory_limit"` // Changed to int64
// 	FileOffset      int64  `bigquery:"file_offset"`  // Changed to int64
// 	Filename        string `bigquery:"filename"`
// 	BuildID         string `bigquery:"build_id"`
// 	HasFunctions    bool   `bigquery:"has_functions"`
// 	HasFilenames    bool   `bigquery:"has_filenames"`
// 	HasLineNumbers  bool   `bigquery:"has_line_numbers"`
// 	HasInlineFrames bool   `bigquery:"has_inline_frames"`
// }

// // BQLocation represents the nested structure for locations
// type BQLocation struct {
// 	ID        int64    `bigquery:"id"`         // Changed to int64
// 	MappingID int64    `bigquery:"mapping_id"` // Changed to int64
// 	Address   int64    `bigquery:"address"`    // Changed to int64
// 	Line      []BQLine `bigquery:"line"`
// 	IsFolded  bool     `bigquery:"is_folded"`
// }

// // BQLine represents the nested structure for lines within a location
// type BQLine struct {
// 	FunctionID int64 `bigquery:"function_id"` // Changed to int64
// 	Line       int64 `bigquery:"line"`
// 	Column     int64 `bigquery:"column"`
// }

// // BQFunction represents the nested structure for functions
// type BQFunction struct {
// 	ID         int64  `bigquery:"id"` // Changed to int64
// 	Name       string `bigquery:"name"`
// 	SystemName string `bigquery:"system_name"`
// 	Filename   string `bigquery:"filename"`
// 	StartLine  int64  `bigquery:"start_line"`
// }

// // BQKeyValue represents the nested structure for key-value pair labels (deployment_labels, profile_labels)
// type BQKeyValue struct {
// 	Key   string `bigquery:"key"`
// 	Value string `bigquery:"value"`
// }

// // Helper to safely get string from table, handling potential out-of-bounds.
// func getString(table []string, index int64) string {
// 	if table == nil || index < 0 || index >= int64(len(table)) {
// 		if index != 0 {
// 			log.Printf("Warning: String table index %d out of bounds (table size %d)", index, len(table))
// 		}
// 		return "" // Return empty string for invalid indices
// 	}
// 	return table[index]
// }

// // --- BigQuery Upload Function ---
// func uploadProfileToBigQuery(ctx context.Context, client *bigquery.Client, profileRow *BigQueryProfileRow) error {
// 	inserter := client.Dataset(datasetID).Table(tableID).Inserter()

// 	items := []*BigQueryProfileRow{profileRow} // Put the single row into a slice for the Inserter

// 	log.Printf("Uploading profile %s to BigQuery table %s.%s.%s...", profileRow.ProfileUUID, projectID, datasetID, tableID)

// 	if err := inserter.Put(ctx, items); err != nil {
// 		log.Printf("Error inserting row into BigQuery: %v", err)
// 		// Attempt to log detailed multi-error if available
// 		if multiErr, ok := err.(bigquery.PutMultiError); ok {
// 			for _, rowErr := range multiErr {
// 				log.Printf("  Row index %d error: %v", rowErr.RowIndex, rowErr.Errors)
// 			}
// 		}
// 		return fmt.Errorf("failed to insert profile into BigQuery: %w", err)
// 	}

// 	log.Printf("Successfully uploaded profile %s.", profileRow.ProfileUUID)
// 	return nil
// }

// func main() {
// 	ctx := context.Background()

// 	// --- Initialize BigQuery Client ---
// 	log.Println("Initializing BigQuery client...")
// 	bqClient, err := bigquery.NewClient(ctx, projectID)
// 	if err != nil {
// 		log.Fatalf("Failed to create BigQuery client: %v", err)
// 	}
// 	defer bqClient.Close()
// 	log.Println("BigQuery client initialized successfully.")

// 	// 1. DECODE BASE64 STRING TO []byte
// 	gzippedBytes, err := base64.StdEncoding.DecodeString(gzippedProfileBytesBase64)
// 	if err != nil {
// 		log.Fatalf("Error decoding base64 string: %v", err)
// 	}
// 	// log.Printf("Successfully decoded base64 string (%d bytes)", len(gzippedBytes))

// 	// 2. DECOMPRESS GZIP
// 	var protoBytes []byte
// 	byteReader := bytes.NewReader(gzippedBytes)
// 	gzipReader, err := gzip.NewReader(byteReader)
// 	if err != nil {
// 		if errors.Is(err, gzip.ErrHeader) {
// 			log.Println("Input doesn't seem to be GZIP compressed. Trying to unmarshal directly.")
// 			protoBytes = gzippedBytes
// 		} else if errors.Is(err, io.ErrUnexpectedEOF) && len(gzippedBytes) < 10 {
// 			log.Printf("Input is very short (%d bytes) and not GZIP compressed. Trying to unmarshal directly.", len(gzippedBytes))
// 			protoBytes = gzippedBytes
// 		} else {
// 			log.Fatalf("Error creating gzip reader: %v", err)
// 		}
// 	} else {
// 		defer gzipReader.Close()
// 		var readErr error
// 		protoBytes, readErr = io.ReadAll(gzipReader)
// 		if readErr != nil {
// 			log.Fatalf("Error reading decompressed gzip data: %v", readErr)
// 		}
// 		// log.Printf("Successfully decompressed gzip data (%d bytes)", len(protoBytes))
// 	}

// 	if len(protoBytes) == 0 {
// 		log.Fatalf("No data available to unmarshal after decoding/decompression attempt.")
// 	}

// 	// 3. UNMARSHAL PROTOBUF
// 	p := &profilepb.Profile{}
// 	if err := proto.Unmarshal(protoBytes, p); err != nil {
// 		log.Fatalf("Error unmarshaling profile proto (%d bytes): %v", len(protoBytes), err)
// 	}
// 	log.Println("Successfully unmarshaled protobuf data")

// 	// --- 4. DENORMALIZATION and POPULATE CONTEXT ---
// 	log.Println("Starting denormalization...")

// 	// Generate a unique ID for this profile upload
// 	profileUUID := uuid.New().String()

// 	// Prepare the target struct
// 	row := BigQueryProfileRow{
// 		// --- CONTEXTUAL / METADATA ---
// 		ProfileUUID:     profileUUID, // Assign generated UUID
// 		UploadTimestamp: time.Now().UTC(),
// 		// Set ProfileType - In real agent, this comes from the outer context (p.GetProfileType().String())
// 		// For this test, let's assume it's CPU or check p.DefaultSampleType? Needs logic.
// 		// Placeholder:
// 		ProfileType: "CPU_TEST", // Replace with actual logic if possible
// 		// Populate deployment/profile labels with placeholder data for testing
// 		DeploymentProjectID: projectID,         // Example: Use current project ID
// 		DeploymentTarget:    "my-test-service", // Example service name
// 		DeploymentLabels: []BQKeyValue{ // Example labels
// 			{Key: "version", Value: "v1.0-test"},
// 			{Key: "env", Value: "local-test"},
// 		},
// 		ProfileLabels: []BQKeyValue{ // Example profile-specific labels
// 			{Key: "instance", Value: "local-instance-1"},
// 		},

// 		// --- DATA FROM PROTOBUF ---
// 		DurationNanos: p.DurationNanos,
// 		Period:        p.Period,
// 	}

// 	// Ensure string table exists and has the mandatory "" at index 0
// 	stringTable := p.StringTable
// 	if stringTable == nil || len(stringTable) == 0 {
// 		// Handle case where StringTable is missing or empty
// 		stringTable = []string{""} // Default to a table with only the empty string
// 	} else if stringTable[0] != "" {
// 		log.Println("Warning: String table index 0 is not empty, prepending empty string.")
// 		stringTable = append([]string{""}, stringTable...)
// 	}

// 	// --- Continue Denormalization using stringTable ---

// 	// Use bigquery.NullTimestamp for time_nanos
// 	if p.TimeNanos != 0 {
// 		// Convert nanoseconds since epoch to time.Time
// 		t := time.Unix(0, p.TimeNanos).UTC()
// 		// Assign to NullTimestamp with Valid = true
// 		row.TimeNanos = bigquery.NullTimestamp{Timestamp: t, Valid: true}
// 		log.Printf("DEBUG: Assigning TimeNanos: %v (Valid=true)", t)
// 	} else {
// 		// Assign to NullTimestamp with Valid = false to represent NULL
// 		row.TimeNanos = bigquery.NullTimestamp{Valid: false}
// 		log.Println("DEBUG: Assigning TimeNanos: (Valid=false)")
// 	}

// 	if p.PeriodType != nil {
// 		row.PeriodType = &BQValueType{
// 			Type: getString(stringTable, p.PeriodType.Type),
// 			Unit: getString(stringTable, p.PeriodType.Unit),
// 		}
// 	}

// 	if len(p.Comment) > 0 {
// 		row.Comment = make([]string, len(p.Comment)) // Optimized allocation
// 		for i, idx := range p.Comment {
// 			row.Comment[i] = getString(stringTable, idx)
// 		}
// 	} else {
// 		row.Comment = []string{} // Ensure non-nil slice for BQ
// 	}

// 	row.DefaultSampleType = getString(stringTable, p.DefaultSampleType)
// 	// Handle potential nil p or missing methods if using older proto versions without Getters
// 	if p != nil {
// 		row.DocURL = getString(stringTable, p.DocUrl) // Direct access if no getter generated
// 		row.DropFramesPattern = getString(stringTable, p.DropFrames)
// 		row.KeepFramesPattern = getString(stringTable, p.KeepFrames)
// 	}

// 	if len(p.SampleType) > 0 {
// 		row.SampleType = make([]BQValueType, len(p.SampleType))
// 		for i, st := range p.SampleType {
// 			row.SampleType[i] = BQValueType{
// 				Type: getString(stringTable, st.Type),
// 				Unit: getString(stringTable, st.Unit),
// 			}
// 		}
// 	} else {
// 		row.SampleType = []BQValueType{} // Ensure non-nil slice
// 	}

// 	// Denormalize Samples
// 	if len(p.Sample) > 0 {
// 		row.Sample = make([]BQSample, len(p.Sample))
// 		for i, s := range p.Sample {
// 			bqSample := BQSample{
// 				// LocationID needs explicit casting
// 				Value: s.Value, // Direct copy ok
// 				Label: make([]BQLabel, 0, len(s.Label)),
// 			}
// 			// Cast LocationID slice
// 			if len(s.LocationId) > 0 {
// 				bqSample.LocationID = make([]int64, len(s.LocationId))
// 				for k, locID := range s.LocationId {
// 					bqSample.LocationID[k] = int64(locID) // Cast element
// 				}
// 			} else {
// 				bqSample.LocationID = []int64{} // Ensure non-nil slice
// 			}

// 			// Value slice check (already correct type)
// 			if s.Value == nil {
// 				bqSample.Value = []int64{}
// 			}

// 			// Label processing (remains the same)
// 			if len(s.Label) > 0 {
// 				bqSample.Label = make([]BQLabel, len(s.Label))
// 				for j, lbl := range s.Label {
// 					bqSample.Label[j] = BQLabel{
// 						Key:     getString(stringTable, lbl.Key),
// 						Str:     getString(stringTable, lbl.Str),
// 						Num:     lbl.Num,
// 						NumUnit: getString(stringTable, lbl.NumUnit),
// 					}
// 				}
// 			} else {
// 				bqSample.Label = []BQLabel{}
// 			}
// 			row.Sample[i] = bqSample
// 		}
// 	} else {
// 		row.Sample = []BQSample{}
// 	}

// 	// Denormalize Mappings
// 	if len(p.Mapping) > 0 {
// 		row.Mapping = make([]BQMapping, len(p.Mapping))
// 		for i, m := range p.Mapping {
// 			row.Mapping[i] = BQMapping{
// 				ID:              int64(m.Id),          // Cast uint64 to int64
// 				MemoryStart:     int64(m.MemoryStart), // Cast uint64 to int64
// 				MemoryLimit:     int64(m.MemoryLimit), // Cast uint64 to int64
// 				FileOffset:      int64(m.FileOffset),  // Cast uint64 to int64
// 				Filename:        getString(stringTable, m.Filename),
// 				BuildID:         getString(stringTable, m.BuildId),
// 				HasFunctions:    m.HasFunctions,
// 				HasFilenames:    m.HasFilenames,
// 				HasLineNumbers:  m.HasLineNumbers,
// 				HasInlineFrames: m.HasInlineFrames,
// 			}
// 		}
// 	} else {
// 		row.Mapping = []BQMapping{}
// 	}

// 	// Denormalize Locations
// 	if len(p.Location) > 0 {
// 		row.Location = make([]BQLocation, len(p.Location))
// 		for i, l := range p.Location {
// 			bqLocation := BQLocation{
// 				ID:        int64(l.Id),        // Cast uint64 to int64
// 				MappingID: int64(l.MappingId), // Cast uint64 to int64
// 				Address:   int64(l.Address),   // Cast uint64 to int64
// 				Line:      make([]BQLine, 0, len(l.Line)),
// 				IsFolded:  l.IsFolded,
// 			}
// 			// Line processing (needs FunctionID cast)
// 			if len(l.Line) > 0 {
// 				bqLocation.Line = make([]BQLine, len(l.Line))
// 				for j, ln := range l.Line {
// 					bqLocation.Line[j] = BQLine{
// 						FunctionID: int64(ln.FunctionId), // Cast uint64 to int64
// 						Line:       ln.Line,
// 						Column:     ln.Column,
// 					}
// 				}
// 			} else {
// 				bqLocation.Line = []BQLine{}
// 			}
// 			row.Location[i] = bqLocation
// 		}
// 	} else {
// 		row.Location = []BQLocation{}
// 	}

// 	// Denormalize Functions
// 	if len(p.Function) > 0 {
// 		row.Function = make([]BQFunction, len(p.Function))
// 		for i, f := range p.Function {
// 			row.Function[i] = BQFunction{
// 				ID:         int64(f.Id), // Cast uint64 to int64
// 				Name:       getString(stringTable, f.Name),
// 				SystemName: getString(stringTable, f.SystemName),
// 				Filename:   getString(stringTable, f.Filename),
// 				StartLine:  f.StartLine,
// 			}
// 		}
// 	} else {
// 		row.Function = []BQFunction{}
// 	}

// 	// Initialize any remaining top-level repeated fields if they were nil
// 	if row.Comment == nil {
// 		row.Comment = []string{}
// 	}
// 	if row.DeploymentLabels == nil {
// 		row.DeploymentLabels = []BQKeyValue{}
// 	}
// 	if row.ProfileLabels == nil {
// 		row.ProfileLabels = []BQKeyValue{}
// 	}

// 	log.Println("Denormalization complete.")

// 	// --- 5. UPLOAD TO BIGQUERY ---
// 	err = uploadProfileToBigQuery(ctx, bqClient, &row)
// 	if err != nil {
// 		// Error is already logged in the upload function
// 		log.Fatalf("Upload failed.") // Exit if upload fails
// 	}

// 	// --- 6. (Optional) PRINT CONFIRMATION ---
// 	fmt.Printf("Successfully processed and uploaded profile with UUID: %s\n", profileUUID)

// 	// --- Original JSON Printing (commented out) ---
// 	// log.Println("Printing denormalized data as JSON:")
// 	// jsonBytes, err := json.MarshalIndent(row, "", "  ")
// 	// if err != nil {
// 	// 	log.Fatalf("Failed to marshal result to JSON: %v", err)
// 	// }
// 	// fmt.Println(string(jsonBytes))
// }
