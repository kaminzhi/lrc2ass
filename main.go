package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type Line struct {
	Time float64
	Text string
}

// 讀取 LRC 行，解析時間跟歌詞
func parseLRCLine(line string) (*Line, error) {
	re := regexp.MustCompile(`\[(\d+):(\d+\.\d+)\](.*)`)
	m := re.FindStringSubmatch(line)
	if len(m) < 4 {
		return nil, fmt.Errorf("invalid line")
	}
	min, _ := strconv.Atoi(m[1])
	sec, _ := strconv.ParseFloat(m[2], 64)
	text := strings.TrimSpace(m[3])
	return &Line{
		Time: float64(min)*60 + sec,
		Text: text,
	}, nil
}

// 格式化 ASS 時間 h:mm:ss.cs
func formatASSTime(sec float64) string {
	h := int(sec) / 3600
	m := (int(sec) % 3600) / 60
	s := int(sec) % 60
	cs := int((sec - float64(int(sec))) * 100)
	return fmt.Sprintf("%d:%02d:%02d.%02d", h, m, s, cs)
}

// 將歌詞拆字並平均分配時間，產生 karaoke 格式 {\kN}字
func karaokeFormat(text string, durationCS int) string {
	runes := []rune(text)
	n := len(runes)
	if n == 0 {
		return ""
	}
	per := durationCS / n
	var sb strings.Builder
	for _, r := range runes {
		sb.WriteString(fmt.Sprintf("{\\k%d}%c", per, r))
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

	scanner := bufio.NewScanner(file)
	var lines []*Line
	for scanner.Scan() {
		line := scanner.Text()
		l, err := parseLRCLine(line)
		if err == nil {
			lines = append(lines, l)
		}
	}

	// ASS header
	header := `[Script Info]
Title: Karaoke Converted from LRC
ScriptType: v4.00+
PlayResX: 384
PlayResY: 288
Timer: 100.0000

[V4+ Styles]
Format: Name, Fontname, Fontsize, PrimaryColour, SecondaryColour, OutlineColour, BackColour, Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment, MarginL, MarginR, MarginV, Encoding
Style: Default,Arial,36,&H00FFFFFF,&H000000FF,&H00000000,&H64000000,0,0,0,0,100,100,0,0,1,1,0,2,10,10,10,1

[Events]
Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text
`

	outf.WriteString(header)

	for i := 0; i < len(lines); i++ {
		start := lines[i].Time
		var end float64
		if i+1 < len(lines) {
			end = lines[i+1].Time
		} else {
			end = start + 5.0 // 最後一句預設 5 秒
		}
		durationCS := int((end - start) * 100) // 百分之一秒
		lyric := karaokeFormat(lines[i].Text, durationCS)
		startStr := formatASSTime(start)
		endStr := formatASSTime(end)
		dialogue := fmt.Sprintf("Dialogue: 0,%s,%s,Default,,0,0,0,,%s\n", startStr, endStr, lyric)
		outf.WriteString(dialogue)
	}

	fmt.Printf("Converted %s to %s done.\n", inputFile, outputFile)
}
