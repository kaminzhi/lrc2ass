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

// Karaoke 文字格式產生器，會用 {\kNN} 標記，NN是每字持續時間(百分之一秒)
func karaokeFormat(text string, durationCS int) string {
	if len(text) == 0 {
		return ""
	}
	// 平均分配每個字的持續時間
	perChar := durationCS / len([]rune(text))
	if perChar <= 0 {
		perChar = 1
	}

	var sb strings.Builder
	for _, r := range text {
		sb.WriteString(fmt.Sprintf("{\\k%d}%c", perChar, r))
	}
	return sb.String()
}

// 移除歌詞中的時間標記 [mm:ss.xx]
var timeTagRe = regexp.MustCompile(`\[\d+:\d+\.\d+\]`)

func cleanLyric(text string) string {
	return timeTagRe.ReplaceAllString(text, "")
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
	var lines []Syllable

	for scanner.Scan() {
		line := scanner.Text()
		sylls, err := parseLine(line)
		if err != nil || len(sylls) == 0 {
			continue
		}
		// LRC 一行只取第一個 syllable，通常 LRC 不會一行多時間
		lines = append(lines, sylls[0])
	}

	// ASS Header，字體大小改成 24 (原本通常是 36 太大)
	header := `[Script Info]
Title: Karaoke Converted from LRC
ScriptType: v4.00+
PlayResX: 384
PlayResY: 288
Timer: 100.0000

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Default,Arial,24,&H00FFFFFF,&H000000FF,&H00000000,&H64000000,0,0,0,0,100,100,0,0,1,1,0,2,10,10,10,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
`
	writer.WriteString(header)

	for i := 0; i < len(lines); i++ {
		start := lines[i].Time
		var end float64
		if i+1 < len(lines) {
			end = lines[i+1].Time
		} else {
			end = start + 5.0
		}
		durationCS := int((end - start) * 100)

		// 移除歌詞中時間標記，再產生 karaoke 格式
		cleanText := cleanLyric(lines[i].Text)
		lyric := karaokeFormat(cleanText, durationCS)

		startStr := formatASSTime(start)
		endStr := formatASSTime(end)

		dialogue := fmt.Sprintf("Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n", startStr, endStr, lyric)
		writer.WriteString(dialogue)
	}

	writer.Flush()

	fmt.Printf("Converted %s to %s successfully.\n", inputFile, outputFile)
}
