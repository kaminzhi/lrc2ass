package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Syllable struct {
	StartTime float64
	EndTime   float64
	Text      string
}

type Line struct {
	Start     float64
	End       float64
	Syllables []Syllable
	Text      string
}

func parseLine(line string) (Line, error) {
	timeTag := regexp.MustCompile(`\[(\d+):(\d+\.\d+)\]`)
	words := timeTag.FindAllStringSubmatchIndex(line, -1)

	if len(words) == 0 {
		return Line{}, nil
	}

	var times []float64
	for _, match := range words {
		min, _ := strconv.Atoi(line[match[2]:match[3]])
		sec, _ := strconv.ParseFloat(line[match[4]:match[5]], 64)
		times = append(times, float64(min)*60+sec)
	}

	textParts := timeTag.Split(line, -1)[1:] // Remove first empty element
	var syllables []Syllable
	for i := 0; i < len(times); i++ {
		var end float64
		if i+1 < len(times) {
			end = times[i+1]
		} else {
			end = times[i] + 1.0 // fallback
		}
		syllables = append(syllables, Syllable{
			StartTime: times[i],
			EndTime:   end,
			Text:      strings.TrimSpace(textParts[i]),
		})
	}

	return Line{
		Start:     syllables[0].StartTime,
		End:       syllables[len(syllables)-1].EndTime,
		Syllables: syllables,
		Text:      strings.Join(textParts, ""),
	}, nil
}

func formatASSTime(sec float64) string {
	h := int(sec) / 3600
	m := (int(sec) % 3600) / 60
	s := int(sec) % 60
	cs := int((sec - float64(int(sec))) * 100)
	return fmt.Sprintf("%d:%02d:%02d.%02d", h, m, s, cs)
}

func formatASS(line Line, nextText string) string {
	var sb strings.Builder
	for _, syl := range line.Syllables {
		dur := syl.EndTime - syl.StartTime
		k := int(dur * 100)
		sb.WriteString(fmt.Sprintf("{\\k%d}%s", k, syl.Text))
	}
	start := formatASSTime(line.Start)
	end := formatASSTime(line.End)
	// 主行
	result := fmt.Sprintf("Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n", start, end, sb.String())
	// 下行
	if nextText != "" {
		result += fmt.Sprintf("Dialogue: 0,%s,%s,NextLine,,0,0,0,,%s\n", start, end, nextText)
	}
	return result
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: lrc2ass input.lrc output.ass")
		return
	}
	infile := os.Args[1]
	outfile := os.Args[2]

	f, err := os.Open(infile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open error:", err)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var lines []Line
	for scanner.Scan() {
		line := scanner.Text()
		parsed, err := parseLine(line)
		if err == nil && parsed.Text != "" {
			lines = append(lines, parsed)
		}
	}

	// 按時間排序
	sort.Slice(lines, func(i, j int) bool {
		return lines[i].Start < lines[j].Start
	})

	out, err := os.Create(outfile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create error:", err)
		return
	}
	defer out.Close()
	writer := bufio.NewWriter(out)

	// Header
	writer.WriteString(`[Script Info]
Title: LRC to ASS
ScriptType: v4.00+
PlayResX: 1280
PlayResY: 720
Timer: 100.0000

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Default,Arial,17,&H00FFFFFF,&H000000FF,&H00000000,&H64000000,-1,0,0,0,100,100,0,0,1,3,0,2,10,10,50,1
Style: NextLine,Arial,15,&H00666666,&H000000FF,&H00000000,&H64000000,0,0,0,0,100,100,0,0,1,1,0,2,10,10,100,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
`)

	for i := 0; i < len(lines); i++ {
		nextText := ""
		if i+1 < len(lines) {
			nextText = lines[i+1].Text
		}
		writer.WriteString(formatASS(lines[i], nextText))
	}

	writer.Flush()
	fmt.Printf("Converted %s to %s\n", infile, outfile)
}
