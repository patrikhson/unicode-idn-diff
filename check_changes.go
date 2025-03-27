package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Regular expression to match Unicode versions (12.0.0 and up)
var unicodeVersionRegex = regexp.MustCompile(`^1[2-9](\.\d+)*$`)

// Collect data that will go in last Appendix
type Entry struct {
	Number int
	Name   string
}

// Reads code point properties from allcodepoints.txt
func readCodepointProperties(filePath string) (map[string]string, map[string]string, error) {
	properties := make(map[string]string)
	codePointNames := make(map[string]string)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ";")
		if len(fields) < 2 {
			continue
		}
		codepoint, property, codePointName := fields[0], fields[1], fields[3]
		properties[codepoint] = property
		codePointNames[codepoint] = codePointName
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}

	return properties, codePointNames, nil
}

// hexToInt converts a hexadecimal string (like "0041") to an integer
func hexToInt(hexStr string) int {
	value, err := strconv.ParseInt(hexStr, 16, 32)
	if err != nil {
		panic(fmt.Sprintf("Invalid hex string: %s", hexStr))
	}
	return int(value)
}

// Reads general category properties from DerivedGeneralCategory.txt
func readGeneralCategory(filePath string) (map[string]string, error) {
	categories := make(map[string]string)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ";")
		if len(fields) < 2 {
			continue
		}
		codepointRange, category := strings.TrimSpace(fields[0]), strings.TrimSpace(fields[1])
		category = strings.TrimSpace(strings.Split(category, "#")[0])
		if strings.Contains(codepointRange, "..") {
			rangeParts := strings.Split(codepointRange, "..")
			start, err1 := strconv.ParseInt(rangeParts[0], 16, 32)
			end, err2 := strconv.ParseInt(rangeParts[1], 16, 32)
			if err1 != nil || err2 != nil {
				continue
			}
			for i := start; i <= end; i++ {
				codepoint := fmt.Sprintf("%04X", i)
				categories[codepoint] = category
			}
		} else {
			codepoint := strings.TrimSpace(codepointRange)
			categories[codepoint] = category
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return categories, nil
}

// Reads NFK data from a file
func readNFKData(filePath string) (map[string][]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	nfkData := make(map[string][]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ";")
		if len(parts) > 1 {
			key := strings.TrimPrefix(parts[0], "U+")
			nfkData[key] = parts[1:] // Store values in the map
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return nfkData, nil
}

func compareVersions(version1, version2 string) {
	// Initialize the slice of entries
	var appendix []Entry
	numAppendixA := 0
	numAppendixB := 0
	numAppendixC := 0
	numAppendixD := 0
	numAppendixE := 0

	// Read properties for the first version
	filePath1 := filepath.Join(version1, "allcodepoints.txt")
	properties1, _, err := readCodepointProperties(filePath1)
	if err != nil {
		fmt.Printf("Error reading %s: %s\n", filePath1, err)
		return
	}

	// Read properties for the second version
	filePath2 := filepath.Join(version2, "allcodepoints.txt")
	properties2, codePointNames2, err := readCodepointProperties(filePath2)
	if err != nil {
		fmt.Printf("Error reading %s: %s\n", filePath2, err)
		return
	}

	// Create a slice to hold the codepoints as integers
	var codepoints []int

	// Populate the slice with the keys (codepoints)
	for codepoint := range properties2 {
		codepoints = append(codepoints, hexToInt(codepoint))
	}

	// Sort the slice of codepoints
	sort.Ints(codepoints)

	// Compare derived property values between the two versions
	var buffer strings.Builder

	fmt.Printf("Comparing version %s and %s\n", version1, version2)
	fmt.Printf("Comparing derived property values\n")

	// Check if the derived property value changed for any code point

	changeCounts := make(map[string]int)

	fmt.Fprintf(&buffer, "\nAppendix A: Code points that changed derived property values\n\n")

	// Iterate through the sorted codepoints
	for _, codepointInt := range codepoints {
		codepoint := fmt.Sprintf("%04X", codepointInt) // Convert back to hex
		oldProperty, existedBefore := properties1[codepoint]
		newProperty := properties2[codepoint]

		// Check if the derived property value changed
		if existedBefore && oldProperty != newProperty {
			changeKey := fmt.Sprintf("%s to %s", oldProperty, newProperty)
			changeCounts[changeKey]++
			// Check if the derived property value changed from UNASSIGNED to something else
			if oldProperty != "UNASSIGNED" {
				fmt.Printf("%s changed from %s to %s\n", codepoint, oldProperty, newProperty)
				if numAppendixA == 0 {
					fmt.Fprintf(&buffer, "# Code point; Old; New; Name\n")
				}
				fmt.Fprintf(&buffer, "U+%s; %s; %s; %s\n", codepoint, oldProperty, newProperty, codePointNames2[codepoint])
				appendix = append(appendix, Entry{codepointInt, fmt.Sprintf("U+%s; UNDER REVIEW # %s", codepoint, codePointNames2[codepoint])})
				numAppendixA++
				numAppendixE++
			}
		}
	}

	if numAppendixA == 0 {
		fmt.Fprintf(&buffer, "# No change in derived property value except from UNASSIGED\n")
	}
	fmt.Printf("Number of code points in Appendix A: %d\n", numAppendixA)

	// Print summary of changes
	fmt.Printf("Count changes in derived property values\n")
	if len(changeCounts) > 0 {
		var totalCount int
		var sortedChanges []string
		totalCount = 0
		for change, count := range changeCounts {
			totalCount += count
			theWord := "points"
			if count == 1 {
				theWord = "point"
			}
			sortedChanges = append(sortedChanges, fmt.Sprintf("# %d code %s changed from %s", count, theWord, change))
		}
		sort.Strings(sortedChanges)
		for _, change := range sortedChanges {
			fmt.Fprintln(&buffer, change)
		}
		theWord := "points"
		if totalCount == 1 {
			theWord = "point"
		}
		fmt.Fprintf(&buffer, "# %d code %s changed in total\n", totalCount, theWord)
	} else {
		fmt.Fprintf(&buffer, "# No derived property changes detected.\n")
	}

	// Check the General_Category
	fmt.Printf("Reading General Category definitions\n")

	// Read the General_Category property for the code points that changed
	generalCategory1, err := readGeneralCategory(filepath.Join(version1, "DerivedGeneralCategory.txt"))
	if err != nil {
		fmt.Printf("Error reading %s: %s\n", filepath.Join(version1, "DerivedGeneralCategory.txt"), err)
		return
	}

	generalCategory2, err := readGeneralCategory(filepath.Join(version2, "DerivedGeneralCategory.txt"))
	if err != nil {
		fmt.Printf("Error reading %s: %s\n", filepath.Join(version2, "DerivedGeneralCategory.txt"), err)
		return
	}

	// Check if the General_Category property changed for any code point
	// Ignore changes if the derived property is UNASSIGNED
	fmt.Printf("Check changes in General Category:\n")
	fmt.Fprintf(&buffer, "\n\nAppendix B: Changes in General Category\n\n")

	// Iterate through the sorted codepoints
	for _, codepointInt := range codepoints {
		codepoint := fmt.Sprintf("%04X", codepointInt) // Convert back to hex
		oldProperty, existedBefore := properties1[codepoint]
		newProperty := properties2[codepoint]
		// Only check if the code point existed in the first version
		if existedBefore {
			oldCategory := generalCategory1[codepoint]
			newCategory := generalCategory2[codepoint]
			// If GC has changed, and the derived property is not UNASSIGNED in both versions
			if oldCategory != newCategory && oldProperty != "UNASSIGNED" && newProperty != "UNASSIGNED" {
				if numAppendixB == 0 {
					fmt.Fprintf(&buffer, "# Code point; Old GC; New GC; Name\n\n")
				}
				fmt.Printf("Code point U+%s changed from %s to %s (General Category: %s to %s)\n",
					codepoint, oldProperty, newProperty, oldCategory, newCategory)
				fmt.Fprintf(&buffer, "U+%s; %s; %s; %s\n", codepoint, oldCategory, newCategory, codePointNames2[codepoint])
				// Should we add to thes code points to UNDER REVIEW, i.e. from PVALID?
				// appendix = append(appendix, Entry{codepointInt, fmt.Sprintf("U+%s; UNDER REVIEW (gc) # %s", codepoint, codePointNames2[codepoint])})
				numAppendixB++
				// numAppendixE++
			}
		}
	}
	if numAppendixB == 0 {
		fmt.Fprintf(&buffer, "# No changes in General Category detected\n")
	}
	fmt.Printf("Number of code points in Appendix B: %d\n", numAppendixB)

	// Check code points that have General_Category Mn
	fmt.Fprintf(&buffer, "\n\nAppendix C: New code points where General Category is Mn\n\n")

	fmt.Printf("Count code points with General_Category Mn\n")

	// Count the number of code points with General_Category Mn in the first version
	count1Mn := 0
	for codepoint, property := range properties1 {
		if property != "UNASSIGNED" && generalCategory1[codepoint] == "Mn" {
			count1Mn++
		}
	}
	fmt.Printf("Number of code points with General_Category Mn in version %s: %d\n", version1, count1Mn)

	// Count the number of code points with General_Category Mn in the second version
	count2Mn := 0
	for codepoint, property := range properties2 {
		if property != "UNASSIGNED" && generalCategory2[codepoint] == "Mn" {
			count2Mn++
		}
	}
	fmt.Printf("Number of code points with General_Category Mn in version %s: %d\n", version2, count2Mn)

	// Check what code points have general category Mn in second version
	for _, codepointInt := range codepoints {
		codepoint := fmt.Sprintf("%04X", codepointInt) // Convert back to hex
		property := properties2[codepoint]
		// Only code points where derived property is not UNASSIGNED, and GC is Mn, in the second version
		if property != "UNASSIGNED" && generalCategory2[codepoint] == "Mn" {
			// Check if the code point did not have General_Category Mn in the first version
			// I.e. skip code points that already had General_Category Mn in the first version
			if generalCategory1[codepoint] != "Mn" {
				if numAppendixC == 0 {
					fmt.Fprintf(&buffer, "# Code point; Name\n")
				}
				fmt.Fprintf(&buffer, "U+%s; %s\n", codepoint, codePointNames2[codepoint])
				appendix = append(appendix, Entry{codepointInt, fmt.Sprintf("U+%s; UNDER REVIEW # %s", codepoint, codePointNames2[codepoint])})
				numAppendixC++
				numAppendixE++
			}
		}
	}
	if numAppendixC == 0 {
		fmt.Fprintf(&buffer, "# No new code points with General Category Mn\n")
	}
	fmt.Printf("Increase in number of code points with General_Category Mn: %d\n", count2Mn-count1Mn)
	fmt.Printf("Number of code points in Appendix C: %d\n", numAppendixC)

	// Check changes in NFK
	fmt.Fprintf(&buffer, "\n\nAppendix D: New code points with NFK normalization\n\n")

	// Read NFK for all code points from the file nfk.txt
	fmt.Printf("\nCheck changes in NFK for all code points\n")

	// Read NFK data for the first version
	nfkPath1 := filepath.Join(version1, "nfk.txt")
	nfk1, err := readNFKData(nfkPath1)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Read NFK data for the second version
	nfkPath2 := filepath.Join(version2, "nfk.txt")
	nfk2, err := readNFKData(nfkPath2)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Iterate through the sorted codepoints and check NFK
	noChangeFromOtherNFK := true
	for _, codepointInt := range codepoints {
		codepoint := fmt.Sprintf("%04X", codepointInt) // Convert back to hex
		oldProperty, existedBefore := properties1[codepoint]
		newProperty := properties2[codepoint]
		// Only check if the code point existed in the first version
		if existedBefore {
			oldNFK := strings.Join(nfk1[codepoint], " ")
			newNFK := strings.Join(nfk2[codepoint], " ")
			// Check if the NFK changed, and the derived property is not UNASSIGNED
			if oldProperty != "UNASSIGNED" && len(nfk2[codepoint]) > 1 && oldNFK != newNFK {
				fmt.Printf("Changed normalization for code point %s (%s %s): %s : %s\n", codepoint, oldProperty, newProperty, oldNFK, newNFK)
				noChangeFromOtherNFK = false
			}
			// Check if the NFK changed from UNASSIGNED to PVALID, and length of NFK is greater than one
			if oldProperty == "UNASSIGNED" && newProperty == "PVALID" && len(nfk2[codepoint]) > 1 {
				fmt.Printf("New code point to normalize %s %s\n", codepoint, newNFK)
				fmt.Fprintf(&buffer, "U+%s; %s; %s\n", codepoint, newNFK, codePointNames2[codepoint])
				appendix = append(appendix, Entry{hexToInt(codepoint), fmt.Sprintf("U+%s; UNDER REVIEW # %s", codepoint, codePointNames2[codepoint])})
				numAppendixD++
				numAppendixE++
			}
		}
	}
	if numAppendixD == 0 {
		fmt.Fprintf(&buffer, "# No new code points with length of NFK greater than one\n")
	}
	if noChangeFromOtherNFK {
		fmt.Println("No change in NFK")
	}
	fmt.Println("Number of new code points with length of NFK greater than one: ", numAppendixD)
	fmt.Println("Number of code points in Appendix D: ", numAppendixD)

	fmt.Fprintf(&buffer, "\nAppendix E: Additions to Exceptions (F)\n\n")
	num2Appendix := 0

	// Sort the appendix by Number
	sort.Slice(appendix, func(i, j int) bool {
		return appendix[i].Number < appendix[j].Number
	})

	// Add the sorted entries to the buffer
	for _, entry := range appendix {
		fmt.Fprintf(&buffer, "%s\n", entry.Name)
		codepoint := fmt.Sprintf("%04X", entry.Number)
		properties2[codepoint] = "UNDER REVIEW"
		num2Appendix++
	}
	fmt.Printf("Total number of entries in Appendix E (Additions to Exceptions): %d\n", numAppendixE)

	if numAppendixE == 0 {
		fmt.Fprintf(&buffer, "# No additional code points to become UNDER REVIEW\n")
	}

	fmt.Fprintf(&buffer, "\nAppendix F: Derived property values Unicode %s\n\n", version2)
	// Loop through the code points and print the derived property values
	// Print the code points in ranges where the property is the same
	var start, end int
	var currentProperty string
	first := true

	for _, codepointInt := range codepoints {
		codepoint := fmt.Sprintf("%04X", codepointInt)
		property := properties2[codepoint]

		if first {
			start = codepointInt
			end = codepointInt
			currentProperty = property
			first = false
		} else if property == currentProperty {
			end = codepointInt
			//fmt.Fprintf(&buffer, "Same property %04X..U+%04X; %s\n", start, end, currentProperty)
		} else {
			if start == end {
				fmt.Fprintf(&buffer, "U+%04X; %s\n", start, currentProperty)
			} else {
				fmt.Fprintf(&buffer, "U+%04X..U+%04X; %s\n", start, end, currentProperty)
			}
			start = codepointInt
			end = codepointInt
			currentProperty = property
		}
	}

	// Print the last range
	if start == end {
		fmt.Fprintf(&buffer, "U+%04X; %s\n", start, currentProperty)
	} else {
		fmt.Fprintf(&buffer, "U+%04X..U+%04X; %s\n", start, end, currentProperty)
	}

	fmt.Print(buffer.String())
	buffer.Reset()
	fmt.Printf("===================\n")
}

func main() {
	// Check if exactly two arguments are provided
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run check_changes.go <version1> <version2>")
		return
	}

	version1 := os.Args[1]
	version2 := os.Args[2]

	// Check if the versions are valid
	if !unicodeVersionRegex.MatchString(version1) || !unicodeVersionRegex.MatchString(version2) {
		fmt.Println("Invalid version format. Please use the format 12.0.0")
		return
	}

	// Call the compareVersions function
	compareVersions(version1, version2)
}
