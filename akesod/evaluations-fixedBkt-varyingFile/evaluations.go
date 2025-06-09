package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/logging/logadmin"
	"cloud.google.com/go/storage"
	"golang.org/x/time/rate"
	"google.golang.org/api/iterator"
)

const (
	baseContent         = "_"
	bucketSize          = 1024 * 1024 // in KB
	projectID           = "ornate-flame-397517"
	region              = "us-east1"
	cloudFunc           = "encrypt-object-1"
	requestsPerMinute   = 50
	sleepDuration       = time.Second * 60 / requestsPerMinute // This seems unused for log fetching
	metadataUpdateTopic = "np-MetadataUpdate"
	uploadKey           = "../keys/updatekey"
	updateKey           = "../keys/updatekey2"
	bucketPrefix        = "fig-11-"
	// New constant for Akeso log query buffer
	akesoLogQueryLookbackDuration = -60 * time.Second // Adjustable: e.g., -20s, -30s. Start with a shorter window.
)

func main() {

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute) // Increased timeout slightly for more operations
	defer cancel()

	objectSizes := map[string]int{
		"128KB": 128,
		"1MB":   1 * 1024,
		"2MB":   2 * 1024,
		"4MB":   4 * 1024,
		"16MB":  16 * 1024,
	}

	strategies := []string{"cmek", "cmek-hsm", "csek", "keywrap", "strawman", "akeso"}

	if len(os.Args) > 1 && os.Args[1] == "prepare" {
		ensureBucketsExist(ctx, objectSizes, strategies)
		GenerateBucketFillingFiles(objectSizes, strategies)
		setSoftDeleteOff(objectSizes, strategies)
		setNotifications(objectSizes)
	} else {
		fmt.Println("Assuming Buckets and Environment is prepared for benchmarking. Run `./evaluation prepare` first if not.")
		CopyFilesToBucket(objectSizes, strategies)
		_ = BenchmarkReencryptions(objectSizes, strategies)
	}
}

func setNotifications(objectSizes map[string]int) {
	var wg sync.WaitGroup
	for sizeKey := range objectSizes {
		wg.Add(1)
		go func(currentSizeKey string) {
			defer wg.Done()
			bucketName := bucketPrefix + strings.ToLower(currentSizeKey) + "-akeso"
			command := fmt.Sprintf("../gcs-utils -notification-config -topic-id %s -project-id %s -event-type OBJECT_METADATA_UPDATE -custom-attributes=new_dek=x++yudvAtO9scc4NPYQusmPD0bsiyQic9ZHziqu62DE= gs://%s", metadataUpdateTopic, projectID, bucketName)
			cmdParts := strings.Fields(command)
			cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
			fmt.Println("Executing Notification Command: ", cmd.String())
			err := cmd.Run()
			if err != nil {
				fmt.Printf("Error adding Notification for %s\nerror:%s\n", bucketName, err)
			} else {
				fmt.Printf("Notification set for %s\n", bucketName)
			}
		}(sizeKey)
	}
	wg.Wait()
}

func setSoftDeleteOff(objectSizes map[string]int, strategies []string) {
	var wg sync.WaitGroup
	for sizeKey := range objectSizes {
		for _, strategy := range strategies {
			wg.Add(1)
			go func(currentSizeKey, currentStrategy string) {
				defer wg.Done()
				bucketName := bucketPrefix + strings.ToLower(currentSizeKey) + "-" + currentStrategy
				command := fmt.Sprintf("gcloud storage buckets update --clear-soft-delete gs://%s", bucketName)
				cmdParts := strings.Fields(command)
				cmd := exec.Command(cmdParts[0], cmdParts[1:]...)
				err := cmd.Run()
				if err != nil {
					fmt.Printf("Error Disabling Soft-Delete for %s: %v\n", bucketName, err)
				}
			}(sizeKey, strategy)
		}
	}
	wg.Wait()
	fmt.Println("Attempted to turn off Soft Delete for all relevant buckets.")
}

func ensureBucketsExist(ctx context.Context, objectSizes map[string]int, strategies []string) {
	var wg sync.WaitGroup
	for sizeKey := range objectSizes {
		for _, strategy := range strategies {
			wg.Add(1)
			go func(sKey, strat string) {
				defer wg.Done()
				bucketName := bucketPrefix + strings.ToLower(sKey) + "-" + strat
				checkAndCreateBucket(ctx, bucketName)
			}(sizeKey, strategy)
		}
	}
	wg.Wait()
}

func checkAndCreateBucket(ctx context.Context, bucketName string) error {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("storage.NewClient: %v", err)
	}
	defer client.Close()
	fmt.Println("Checking bucket Handle: ", bucketName)
	bucket := client.Bucket(bucketName)

	_, err = bucket.Attrs(ctx)
	if err == nil {
		fmt.Println(bucketName, " found. Deleting and recreating...")
		it := bucket.Objects(ctx, nil)
		for {
			attrs, err := it.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				fmt.Printf("Error listing objects in %s for deletion: %v\n", bucketName, err)
				return fmt.Errorf("Bucket(%q).Objects: %v", bucketName, err)
			}
			if err := bucket.Object(attrs.Name).Delete(ctx); err != nil {
				fmt.Printf("Error deleting object %s in %s: %v\n", attrs.Name, bucketName, err)
			}
		}

		err := bucket.Delete(ctx)
		if err != nil {
			fmt.Printf("Bucket %s deleting error: %v\n", bucketName, err)
		} else {
			fmt.Printf("Bucket %s deleted\n", bucketName)
		}
		time.Sleep(2 * time.Second) // Give GCP time to process deletion fully
	}

	bucketAttrs := &storage.BucketAttrs{
		Location:               region,
		PublicAccessPrevention: storage.PublicAccessPreventionEnforced,
		UniformBucketLevelAccess: storage.UniformBucketLevelAccess{
			Enabled: true,
		},
		VersioningEnabled: false,
	}
	err = bucket.Create(ctx, projectID, bucketAttrs)
	if err != nil {
		fmt.Printf("Bucket %s creation error: %v\n", bucketName, err)
		return fmt.Errorf("bucket.Create: %v", err)
	}
	fmt.Printf("Bucket %s created\n", bucketName)
	return nil
}

func CopyFilesToBucket(objectSizes map[string]int, strategies []string) {
	uploadLimiter := rate.NewLimiter(rate.Every(200*time.Millisecond), 1)

	for sizeKey, sizeInKB := range objectSizes {
		numFiles := bucketSize / sizeInKB
		if numFiles == 0 && bucketSize > 0 && sizeInKB > 0 {
			numFiles = 1
		}
		bucketDir := "generated_files/"
		fmt.Printf("Starting upload for size %s (%dKB): %d files\n", sizeKey, sizeInKB, numFiles)

		for i := 1; i <= numFiles; i++ {
			fileName := fmt.Sprintf("%dKB-%d.txt", sizeInKB, i)

			for _, strategy := range strategies {
				maxRetries := 3
				var lastErr error

				for attempt := 0; attempt < maxRetries; attempt++ {
					if err := uploadLimiter.Wait(context.Background()); err != nil {
						fmt.Printf("Rate limiter error: %v\n", err)
						continue
					}

					bucketName := bucketPrefix + strings.ToLower(sizeKey) + "-" + strategy
					var cmd *exec.Cmd
					sourceFilePath := filepath.Join(bucketDir, fileName)
					gcsPath := fmt.Sprintf("gs://%s/%s", bucketName, fileName)

					switch strategy {
					case "cmek":
						command := fmt.Sprintf("../cloud-cp -strategy cmek -cmekKey projects/%s/locations/%s/keyRings/akeso_dev/cryptoKeys/key1 %s %s", projectID, region, sourceFilePath, gcsPath)
						cmdParts := strings.Fields(command)
						cmd = exec.Command(cmdParts[0], cmdParts[1:]...)
					case "cmek-hsm":
						command := fmt.Sprintf("../cloud-cp -strategy cmek -cmekKey projects/%s/locations/%s/keyRings/akeso_dev/cryptoKeys/key2 %s %s", projectID, region, sourceFilePath, gcsPath)
						cmdParts := strings.Fields(command)
						cmd = exec.Command(cmdParts[0], cmdParts[1:]...)
					default:
						cmd = exec.Command("../cloud-cp", "-strategy", strategy, "-key", uploadKey, sourceFilePath, gcsPath)
					}

					output, err := cmd.CombinedOutput()
					if err == nil {
						fmt.Printf("Successfully uploaded %s to %s (%s)\n", fileName, strategy, bucketName)
						lastErr = nil
						break
					}

					lastErr = fmt.Errorf("error executing command for %s to %s: %v\nOutput: %s", fileName, gcsPath, err, string(output))
					if attempt < maxRetries-1 {
						retryDelay := time.Duration(2<<attempt) * time.Second
						fmt.Printf("Upload failed for %s (attempt %d/%d), retrying in %v...\n", fileName, attempt+1, maxRetries, retryDelay)
						time.Sleep(retryDelay)
					}
				}
				if lastErr != nil {
					fmt.Printf("Failed all retries for %s with strategy %s: %v\n", fileName, strategy, lastErr)
					return
				}
			}
			if i%10 == 0 || i == numFiles {
				fmt.Printf("Progress for size %s: %d/%d files uploaded across strategies\n", sizeKey, i, numFiles)
			}
		}
		fmt.Printf("Completed copying all files for size %s\n", sizeKey)
	}
}

func getMinFromValid(times []int64, count int) (int64, bool) {
	if count == 0 {
		return 0, false
	}
	var minVal int64
	initialized := false
	for i := 0; i < count; i++ {
		if times[i] != 0 {
			if !initialized {
				minVal = times[i]
				initialized = true
			} else if times[i] < minVal {
				minVal = times[i]
			}
		}
	}
	return minVal, initialized
}

func getMaxFromValid(times []int64, count int) (int64, bool) {
	if count == 0 {
		return 0, false
	}
	var maxVal int64
	initialized := false
	for i := 0; i < count; i++ {
		if times[i] != 0 {
			if !initialized {
				maxVal = times[i]
				initialized = true
			} else if times[i] > maxVal {
				maxVal = times[i]
			}
		}
	}
	return maxVal, initialized
}

func BenchmarkReencryptions(objectSizes map[string]int, strategies []string) string {
	header := "#ObjectSizes"
	for _, strategy := range strategies {
		header += "\t" + strategy
	}

	datFile := "update-" + strings.Join(strategies, "-") + ".dat"
	file, err := os.Create(datFile)
	if err != nil {
		fmt.Printf("Error creating DAT file: %v\n", err)
		return ""
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	_, _ = writer.WriteString(header + "\n")
	writer.Flush()

	for sizeKey, sizeInKB := range objectSizes {
		sizeReencTimes := make(map[string]float64)
		numFiles := bucketSize / sizeInKB
		if numFiles == 0 && bucketSize > 0 && sizeInKB > 0 {
			numFiles = 1
		}

		for _, strategy := range strategies {
			logFileTracker := make(map[string]bool)
			totalTimeS := 0.0
			bucketName := bucketPrefix + strings.ToLower(sizeKey) + "-" + strategy

			akesoStartTimes := make([]int64, numFiles)
			akesoEndTimes := make([]int64, numFiles)
			akesoFoundFiles := 0

			// MODIFICATION POINT 1: Determine log query start time precisely for Akeso
			var akesoQueryStartTimeForFilter time.Time
			if strategy == "akeso" {
				// This time is captured *before* the loop that executes cloud-cp commands for the current set.
				// The lookback is to account for potential small delays in log ingestion or clock skews.
				// It should be short enough not to pick up logs from unrelated prior benchmark sets.
				effectiveStartOfOperations := time.Now()
				akesoQueryStartTimeForFilter = effectiveStartOfOperations.Add(akesoLogQueryLookbackDuration)
				fmt.Printf("Akeso: Log query filter will target logs after: %s (for %s, %s)\n",
					akesoQueryStartTimeForFilter.Format(time.RFC3339), sizeKey, strategy)
			}

			fmt.Printf("Benchmarking strategy: %s, size: %s (%d files)\n", strategy, sizeKey, numFiles)

			for i := 1; i <= numFiles; i++ {
				sizeBasedFileName := fmt.Sprintf("%dKB-%d.txt", sizeInKB, i)
				var cmd *exec.Cmd
				gcsObjectURL := fmt.Sprintf("gs://%s/%s", bucketName, sizeBasedFileName)

				switch strategy {
				case "cmek":
					command := fmt.Sprintf("../cloud-cp -strategy cmek -cmekKey projects/%s/locations/%s/keyRings/akeso_dev/cryptoKeys/key1/cryptoKeyVersions/1 -cmekUpdateKey projects/%s/locations/%s/keyRings/akeso_dev/cryptoKeys/key3 %s", projectID, region, projectID, region, gcsObjectURL)
					cmdParts := strings.Fields(command)
					cmd = exec.Command(cmdParts[0], cmdParts[1:]...)
				case "cmek-hsm":
					command := fmt.Sprintf("../cloud-cp -strategy cmek -cmekKey projects/%s/locations/%s/keyRings/akeso_dev/cryptoKeys/key2/cryptoKeyVersions/1 -cmekUpdateKey projects/%s/locations/%s/keyRings/akeso_dev/cryptoKeys/key4 %s", projectID, region, projectID, region, gcsObjectURL)
					cmdParts := strings.Fields(command)
					cmd = exec.Command(cmdParts[0], cmdParts[1:]...)
				default:
					cmd = exec.Command("../cloud-cp", "-strategy", strategy, "-maxReenc", "50", "-key", uploadKey, "-updateKey", updateKey, "-dekOverride", "dek.bin", gcsObjectURL)
				}

				output, errCmd := cmd.CombinedOutput()
				if errCmd != nil {
					fmt.Printf("Error executing update command for %s (strategy %s): %v\nCommand was: %s\nOutput: %s\n", sizeBasedFileName, strategy, errCmd, cmd.String(), string(output))
					if strategy != "akeso" {
						continue
					}
				}
				if strategy != "akeso" || errCmd != nil {
					fmt.Printf("Output for %s (%s):\n%s\n", sizeBasedFileName, strategy, string(output))
				}

				if strategy != "akeso" {
					elapsedMs, _, _, errParse := parseTimeOutput(string(output), sizeBasedFileName, strategy)
					if errParse != nil {
						fmt.Printf("Failed to parse times from cloud-cp output for %s (%s): %v\n", sizeBasedFileName, strategy, errParse)
						continue
					}
					totalTimeS += elapsedMs / 1000.0
				}
			}

			if strategy != "akeso" {
				sizeReencTimes[strategy] = totalTimeS
			} else {
				client, errLogClient := logadmin.NewClient(context.Background(), projectID)
				if errLogClient != nil {
					fmt.Printf("Failed to create log client for Akeso: %v\n", errLogClient)
					sizeReencTimes[strategy] = -1
					continue
				}
				defer client.Close()

				maxLogRetries := 5
				logRetryCount := 0

			logQueryLoop:
				for logRetryCount < maxLogRetries {
					// MODIFICATION POINT 2: Use the more precise akesoQueryStartTimeForFilter in the log query
					time.Sleep(sleepDuration)
					fmt.Printf("Akeso: Fetching logs attempt %d/%d (Querying for logs after: %s)\n", logRetryCount+1, maxLogRetries, akesoQueryStartTimeForFilter.Format(time.RFC3339))
					iter := client.Entries(context.Background(),
						logadmin.Filter(fmt.Sprintf(
							`resource.type = "cloud_run_revision"
                            resource.labels.service_name = "%s"
                            resource.labels.location = "%s"
                            severity>=DEFAULT
                            timestamp > "%s"`, //Ensure this uses the precise start time
							cloudFunc, region, akesoQueryStartTimeForFilter.Format(time.RFC3339))),
						logadmin.NewestFirst())

					newLogsFoundThisAttempt := false
					for {
						entry, errIter := iter.Next()
						if errIter == iterator.Done {
							break
						}
						if errIter != nil {
							fmt.Printf("Error reading log entry for Akeso: %v\n", errIter)
							break
						}

						payloadStr, ok := entry.Payload.(string)
						if !ok {
							continue
						}

						if strings.Contains(payloadStr, "[ENC]") {
							parts := strings.Fields(payloadStr)
							if len(parts) < 10 {
								fmt.Printf("Akeso: Invalid log entry format (too few parts): %s\n", payloadStr)
								continue
							}
							fileNameFromLog := parts[3]

							// *** FIX 4: Ensure filename from log matches expected pattern for current sizeKey ***
							// This helps filter out logs from other sizes even if the time window overlaps slightly.
							expectedFilePrefix := fmt.Sprintf("%dKB-", sizeInKB)
							if !strings.HasPrefix(fileNameFromLog, expectedFilePrefix) {
								// This log is not for the current object size being benchmarked.
								// This can happen if akesoQueryStartTimeForFilter is not perfectly isolating.
								// fmt.Printf("Akeso: Skipping log for %s as it does not match current size key prefix %s\n", fileNameFromLog, expectedFilePrefix)
								continue
							}

							if !logFileTracker[fileNameFromLog] {
								if akesoFoundFiles < numFiles {
									logFileTracker[fileNameFromLog] = true
									newLogsFoundThisAttempt = true

									fmt.Printf("Akeso: Found new log for file: %s\n", fileNameFromLog)
									fmt.Printf("Akeso: Log entry: %s\n", payloadStr)

									endTimeStr := strings.TrimSuffix(parts[len(parts)-1], "ns")
									endTime, errParseEnd := strconv.ParseInt(endTimeStr, 10, 64)
									if errParseEnd != nil {
										fmt.Printf("Akeso: Error parsing end time for %s: %v (from '%s')\n", fileNameFromLog, errParseEnd, endTimeStr)
									} else {
										akesoEndTimes[akesoFoundFiles] = endTime
										fmt.Printf("Akeso: Parsed end time for %s: %d\n", fileNameFromLog, endTime)
									}

									startTimeFoundInLog := false
									for k, part := range parts {
										if strings.ToLower(part) == "from" && k+1 < len(parts) {
											startTimeStr := strings.TrimSuffix(parts[k+1], "ns")
											startTime, errParseStart := strconv.ParseInt(startTimeStr, 10, 64)
											if errParseStart != nil {
												fmt.Printf("Akeso: Error parsing start time for %s: %v (from '%s')\n", fileNameFromLog, errParseStart, startTimeStr)
											} else {
												akesoStartTimes[akesoFoundFiles] = startTime
												fmt.Printf("Akeso: Parsed start time for %s: %d\n", fileNameFromLog, startTime)
												startTimeFoundInLog = true
											}
											break
										}
									}
									if !startTimeFoundInLog {
										fmt.Printf("Akeso: Start time token 'from' not found for %s in log entry.\n", fileNameFromLog)
									}
									akesoFoundFiles++
								} else {
									fmt.Printf("Akeso: Warning - Found log for %s but already processed %d files (expected %d).\n", fileNameFromLog, akesoFoundFiles, numFiles)
								}
							}
						}
					}

					if akesoFoundFiles == numFiles {
						fmt.Printf("Akeso: Successfully found logs for all %d files.\n", numFiles)
						break logQueryLoop
					}

					if !newLogsFoundThisAttempt {
						logRetryCount++
						fmt.Printf("Akeso: No new logs found in this query attempt (%d/%d). Total found so far: %d/%d. Waiting before next full query...\n", logRetryCount, maxLogRetries, akesoFoundFiles, numFiles)
						if logRetryCount < maxLogRetries {
							time.Sleep(15 * time.Second) // Consider making this an exponential backoff
						}
					} else {
						fmt.Printf("Akeso: New logs were found in this attempt. Total found so far: %d/%d.\n", akesoFoundFiles, numFiles)
						if akesoFoundFiles < numFiles && logRetryCount < maxLogRetries-1 {
							time.Sleep(5 * time.Second)
						}
					}
				}

				if akesoFoundFiles < numFiles {
					fmt.Printf("Akeso: Process completed for size %s, but only found logs for %d out of %d expected files. Timing will be based on incomplete data or marked as -1.\n", sizeKey, akesoFoundFiles, numFiles)
				} else {
					fmt.Printf("Akeso: Successfully processed logs for all %d files for size %s.\n", numFiles, sizeKey)
				}

				if akesoFoundFiles > 0 {
					minActualStart, okMin := getMinFromValid(akesoStartTimes, akesoFoundFiles)
					maxActualEnd, okMax := getMaxFromValid(akesoEndTimes, akesoFoundFiles)

					if okMin && okMax {
						if maxActualEnd > minActualStart {
							durationNs := maxActualEnd - minActualStart
							durationSeconds := float64(durationNs) / 1e9
							fmt.Printf("Akeso strategy for bucket %s using %s objects:\n", sizeKey, sizeKey)
							fmt.Printf("  Using timing data from %d/%d files.\n", akesoFoundFiles, numFiles)
							fmt.Printf("  Min Actual Start: %d ns, Max Actual End: %d ns\n", minActualStart, maxActualEnd)
							fmt.Printf("  Calculated Duration (s): %.6f\n", durationSeconds)
							sizeReencTimes[strategy] = durationSeconds
							// MODIFICATION POINT 3: If files are missing due to quota, result should reflect that.
							// The DAT file will show the calculated time, but the console log above indicates if data was partial.
							// Consider adding a flag or special value to DAT if data is known to be incomplete.
							// For now, it calculates based on what's found.
						} else {
							fmt.Printf("  Error: Akeso Max Actual End time (%d) is not greater than Min Actual Start time (%d) for size %s. Found %d files.\n", maxActualEnd, minActualStart, sizeKey, akesoFoundFiles)
							sizeReencTimes[strategy] = -1
						}
					} else {
						fmt.Printf("  Error: Akeso could not determine valid min start or max end times from %d found files for size %s.\n", akesoFoundFiles, sizeKey)
						sizeReencTimes[strategy] = -1
					}
				} else {
					fmt.Printf("Akeso: Warning - No Akeso log entries processed for size %s. Cannot calculate duration.\n", sizeKey)
					sizeReencTimes[strategy] = -1
				}
			}
			fmt.Printf("Processed strategy %s for object size %s.\n", strategy, sizeKey)
		}

		datLine := sizeKey
		for _, strategy := range strategies {
			datLine += fmt.Sprintf("\t%.3f", sizeReencTimes[strategy])
		}
		_, err = writer.WriteString(datLine + "\n")
		if err != nil {
			fmt.Printf("Error writing to DAT file for size %s: %v\n", sizeKey, err)
		}
		err = writer.Flush()
		if err != nil {
			fmt.Printf("Error flushing to DAT file for size %s: %v\n", sizeKey, err)
		}
		fmt.Printf("Results for object size %s written to %s\n", sizeKey, filepath.Base(datFile))
	}

	fmt.Printf("Benchmark complete. Results saved to %s\n", filepath.Base(datFile))
	return datFile
}

func parseTimeOutput(output, objectName, strategy string) (elapsedMs, startMs, endMs float64, err error) { // startMs, endMs are not used for non-Akeso
	lines := strings.Split(output, "\n")
	found := false
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		fields := strings.Fields(trimmedLine)

		if len(fields) == 0 {
			continue
		}

		// Handle non-Akeso strategies (cmek, csek, strawman, etc.)
		// Expected format: "OBJECT_NAME TIME_VALUEms"
		// Example: "2048KB-1.txt 321.930342ms"
		if strategy != "akeso" {
			// Check if the line starts with the objectName and has at least one more field for time
			if len(fields) >= 2 && fields[0] == objectName {
				elapsedMs, err = timeStrToMs(fields[1])
				if err != nil {
					// More specific error if time parsing fails for the matched line
					return 0, 0, 0, fmt.Errorf("parsing time for object '%s' from value '%s' in line '%s': %v", objectName, fields[1], trimmedLine, err)
				}
				found = true
				// Return 0 for startMs, endMs as they are not typically parsed from simple non-Akeso CLI outputs
				return elapsedMs, 0, 0, nil
			}
		} else {
			// Akeso timing is primarily handled by Google Cloud Log fetching, so just continue.
			continue
		}
	}

	if !found { // Only set general error if no specific parsing attempt succeeded.
		err = fmt.Errorf("relevant time output not found for object %s (strategy %s) in provided output. Raw output lines checked: %d", objectName, strategy, len(lines))
	}
	return 0, 0, 0, err
}

func timeStrToMs(timeVal string) (timeMs float64, err error) {
	trimmedVal := timeVal
	multiplier := 1.0

	if strings.HasSuffix(trimmedVal, "ns") {
		trimmedVal = strings.TrimSuffix(trimmedVal, "ns")
		multiplier = 1.0 / 1e6
	} else if strings.HasSuffix(trimmedVal, "µs") {
		trimmedVal = strings.TrimSuffix(trimmedVal, "µs")
		multiplier = 1.0 / 1e3
	} else if strings.HasSuffix(trimmedVal, "us") {
		trimmedVal = strings.TrimSuffix(trimmedVal, "us")
		multiplier = 1.0 / 1e3
	} else if strings.HasSuffix(trimmedVal, "ms") {
		trimmedVal = strings.TrimSuffix(trimmedVal, "ms")
		multiplier = 1.0
	} else if strings.HasSuffix(trimmedVal, "s") {
		trimmedVal = strings.TrimSuffix(trimmedVal, "s")
		multiplier = 1e3
	} else {
		val, errConv := strconv.ParseFloat(trimmedVal, 64)
		if errConv == nil {
			return val, nil
		}
		return 0, fmt.Errorf("unknown time units or unparseable: %s", timeVal)
	}

	val, err := strconv.ParseFloat(trimmedVal, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse time value '%s' (from '%s'): %v", trimmedVal, timeVal, err)
	}

	timeMs = val * multiplier
	return timeMs, nil
}

func createFile(sizeKB, fileIndex int, baseDir string) error {
	fileName := fmt.Sprintf("%dKB-%d.txt", sizeKB, fileIndex)
	filePath := filepath.Join(baseDir, fileName)

	if _, err := os.Stat(filePath); err == nil {
		// fmt.Printf("File already exists, skipping: %s\n", fileName) // Make this conditional/less verbose
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("error checking file existence %s: %v", fileName, err)
	}

	contentBuilder := strings.Builder{}
	targetSize := sizeKB * 1024
	if targetSize == 0 {
		return fmt.Errorf("cannot create file of size 0 for %s", fileName)
	}

	baseLen := len(baseContent)
	if baseLen == 0 {
		return fmt.Errorf("baseContent cannot be empty for file %s", fileName)
	}

	numRepeats := targetSize / baseLen
	remainder := targetSize % baseLen

	for k := 0; k < numRepeats; k++ {
		contentBuilder.WriteString(baseContent)
	}
	if remainder > 0 {
		contentBuilder.WriteString(baseContent[:remainder])
	}
	content := contentBuilder.String()

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating file %s: %v", fileName, err)
	}
	defer file.Close()

	_, err = file.WriteString(content)
	if err != nil {
		return fmt.Errorf("error writing to file %s: %v", fileName, err)
	}
	return nil
}

func GenerateBucketFillingFiles(objectSizes map[string]int, strategies []string) {
	var wg sync.WaitGroup
	outputDir := "generated_files"
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		fmt.Printf("Error creating directory '%s': %v\n", outputDir, err)
		os.Exit(1)
	}

	for sizeKey, sizeInKB := range objectSizes {
		numFiles := bucketSize / sizeInKB
		if numFiles == 0 && bucketSize > 0 && sizeInKB > 0 {
			numFiles = 1
		}
		fmt.Printf("For Bucket Size configuration of %dKB: for object size %s (%dKB), files required are: %d\n", bucketSize, sizeKey, sizeInKB, numFiles)

		for num := 1; num <= numFiles; num++ {
			wg.Add(1)
			go func(sKB, n int) {
				defer wg.Done()
				if err := createFile(sKB, n, outputDir); err != nil {
					fmt.Printf("Error creating file (size %dKB, index %d): %v\n", sKB, n, err)
				}
			}(sizeInKB, num)
		}
	}
	wg.Wait()
	fmt.Println("File generation complete.")
}
