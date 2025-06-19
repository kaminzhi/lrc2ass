package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type Syllable struct {
	Time float64
	Text string
}

// 解析一行 lrc，支援 syllable 和普通格式
func parseLine(line string) ([]Syllable, error) {
	if strings.Contains(line, "<") {
		re := regexp.MustCompile(`<(\d+):(\d+\.\d+)> ?([^<]+)`)
		matches := re.FindAllStringSubmatch(line, -1)
		var result []Syllable
		for _, match := range matches {
			min, _ := strconv.Atoi(match[1])
			sec, _ := strconv.ParseFloat(match[2], 64)
			text := match[3]
			result = append(result, Syllable{
				Time: float64(min)*60 + sec,
				Text: strings.TrimSpace(text),
			})
		}
		return result, nil
	}

	// 普通 LRC 格式: [mm:ss.xx] text
	re := regexp.MustCompile(`\[(\d+):(\d+\.\d+)\]\s*(.+)`)
	match := re.FindStringSubmatch(line)
	if len(match) == 4 {
		min, _ := strconv.Atoi(match[1])
		sec, _ := strconv.ParseFloat(match[2], 64)
		text := match[3]
		return []Syllable{
			{Time: float64(min)*60 + sec, Text: text},
		}, nil
	}

	return nil, nil
}

// 格式化 ASS 時間格式: h:mm:ss.cs
func formatASSTime(sec float64) string {
	h := int(sec) / 3600
	m := (int(sec) % 3600) / 60
	s := int(sec) % 60
	cs := int((sec - float64(int(sec))) * 100)
	return fmt.Sprintf("%d:%02d:%02d.%02d", h, m, s, cs)
}

// 產生 ASS Dialogue 字串
func formatASS(syllables []Syllable, nextStart float64) string {
	var sb strings.Builder
	for i := 0; i < len(syllables); i++ {
		start := formatASSTime(syllables[i].Time)
		var end string
		if i+1 < len(syllables) {
			end = formatASSTime(syllables[i+1].Time)
		} else {
			endTime := syllables[i].Time + 0.5
			if nextStart > syllables[i].Time {
				endTime = nextStart
			}
			end = formatASSTime(endTime)
		}
		text := syllables[i].Text
		sb.WriteString(fmt.Sprintf("Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n", start, end, text))
	}
	return sb.String()
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: lrc2ass input.lrc output.ass")
		return
	}
	inputFile := os.Args[1]
	outputFile := os.Args[2]

	file, err := os.Open(inputFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Open input file error:", err)
		return
	}
	defer file.Close()

	outf, err := os.Create(outputFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Create output file error:", err)
		return
	}
	defer outf.Close()

	writer := bufio.NewWriter(outf)

	scanner := bufio.NewScanner(file)
	var allLines [][]Syllable

	for scanner.Scan() {
		line := scanner.Text()
		syllables, err := parseLine(line)
		if err != nil || len(syllables) == 0 {
			continue
		}
		allLines = append(allLines, syllables)
	}

	// ASS header
	header := `[Script Info]
Title: Converted from LRC
ScriptType: v4.00+
PlayResX: 384
PlayResY: 288
Timer: 100.0000

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Default,Arial,20,&H00FFFFFF,&H000000FF,&H00000000,&H64000000,0,0,0,0,100,100,0,0,1,1,0,2,10,10,10,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
`
	writer.WriteString(header)

	for i := 0; i < len(allLines); i++ {
		var nextStart float64
		if i+1 < len(allLines) {
			nextStart = allLines[i+1][0].Time
		}
		writer.WriteString(formatASS(allLines[i], nextStart))
	}

	writer.Flush()

	fmt.Printf("Converted %s to %s successfully.\n", inputFile, outputFile)
}
