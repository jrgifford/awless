package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/wallix/awless/console"
	"github.com/wallix/awless/template"
)

type logPrinter interface {
	print(*template.TemplateExecution) error
}

type fullLogPrinter struct {
	w io.Writer
}

func (p *fullLogPrinter) print(t *template.TemplateExecution) error {
	writeSimpleLogHeader(t, p.w)

	for _, cmd := range t.CommandNodesIterator() {
		var status string
		if cmd.CmdErr != nil {
			status = renderRedFn("KO")
		} else {
			status = renderGreenFn("OK")
		}

		var line string
		if v, ok := cmd.CmdResult.(string); ok && v != "" {
			line = fmt.Sprintf("    %s\t%s\t[%s]", status, cmd.String(), v)
		} else {
			line = fmt.Sprintf("    %s\t%s", status, cmd.String())
		}

		fmt.Fprintln(p.w, line)

		writeError(cmd.Err(), p.w)
	}
	return nil
}

type statLogPrinter struct {
	w io.Writer
}

func (p *statLogPrinter) print(t *template.TemplateExecution) error {
	writeRichLogHeader(t, p.w)

	fmt.Fprintln(p.w)

	if stats := t.Stats(); stats.CmdCount > 1 {
		for _, line := range alignActionEntityCount(stats.ActionEntityCount) {
			fmt.Fprintf(p.w, "\t%s\n", line)
		}
	}
	return nil
}

type shortLogPrinter struct {
	w io.Writer
}

func (p *shortLogPrinter) print(t *template.TemplateExecution) error {
	writeRichLogHeader(t, p.w)
	return nil
}

func newDefaultTemplatePrinter(w io.Writer) logPrinter {
	return &defaultPrinter{w}
}

type defaultPrinter struct {
	w io.Writer
}

func (p *defaultPrinter) print(t *template.TemplateExecution) error {
	for _, cmd := range t.CommandNodesIterator() {
		var status string
		if cmd.Err() != nil {
			status = renderRedFn("KO")
		} else {
			status = renderGreenFn("OK")
		}

		var line string
		if v, ok := cmd.Result().(string); ok && v != "" {
			line = fmt.Sprintf("    %s\t%s = %s\t", status, cmd.Entity, v)
		} else {
			line = fmt.Sprintf("    %s\t%s %s\t", status, cmd.Action, cmd.Entity)
		}

		fmt.Fprintln(p.w, line)
		writeError(cmd.Err(), p.w)
	}
	return nil
}

type rawJSONPrinter struct {
	w io.Writer
}

func (p *rawJSONPrinter) print(t *template.TemplateExecution) error {
	if err := json.NewEncoder(p.w).Encode(t); err != nil {
		return fmt.Errorf("json printer: %s", err)
	}
	return nil
}

type idOnlyPrinter struct {
	w io.Writer
}

func (p *idOnlyPrinter) print(t *template.TemplateExecution) error {
	fmt.Fprint(p.w, t.ID)
	return nil
}

func writeRichLogHeader(t *template.TemplateExecution, w io.Writer) {
	stats := t.Stats()

	fmt.Fprint(w, renderYellowFn(t.ID))
	if stats.KOCount == 0 {
		color.New(color.FgGreen).Fprint(w, " OK")
	} else {
		color.New(color.FgRed).Fprint(w, " KO")
	}

	fmt.Fprint(w, " - ")

	if total := stats.CmdCount; total == 1 {
		fmt.Fprint(w, stats.Oneliner)
	} else {
		fmt.Fprintf(w, "%d commands", total)
	}

	fmt.Fprintf(w, " (%s ago)", console.HumanizeTime(t.Date()))

	if t.Author != "" {
		fmt.Fprintf(w, " by %s", renderBlueFn(t.Author))
	}
	if t.Profile != "" {
		fmt.Fprintf(w, " with profile %s", renderBlueFn(t.Profile))
	}
	if t.Locale != "" {
		fmt.Fprintf(w, " in %s", renderBlueFn(t.Locale))
	}
	if !template.IsRevertible(t.Template) {
		fmt.Fprintf(w, " (not revertible)")
	}
}

func writeSimpleLogHeader(t *template.TemplateExecution, w io.Writer) {
	fmt.Fprintf(w, "ID: %s\tDate: %s", renderYellowFn(t.ID), t.Date().Format(time.Stamp))
	if t.Author != "" {
		fmt.Fprintf(w, "\tAuthor: %s", t.Author)
	}
	if t.Locale != "" {
		fmt.Fprintf(w, "\tRegion: %s", t.Locale)
	}
	if t.Profile != "" {
		fmt.Fprintf(w, "\tProfile: %s", t.Profile)
	}
	if !template.IsRevertible(t.Template) {
		fmt.Fprintf(w, " (not revertible)")
	}
	fmt.Fprintln(w)
}

func writeError(err error, w io.Writer) {
	if err != nil {
		for _, msg := range formatMultiLineErrMsg(err.Error()) {
			fmt.Fprintln(w, renderRedFn(fmt.Sprintf("\t%s", msg)))
		}
	}
}

func formatMultiLineErrMsg(msg string) []string {
	notabs := strings.Replace(msg, "\t", "", -1)
	var indented []string
	for _, line := range strings.Split(notabs, "\n") {
		indented = append(indented, fmt.Sprintf("    %s", line))
	}
	return indented
}

func alignActionEntityCount(items map[string]int) (out []string) {
	var all []string
	for actionentity, count := range items {
		all = append(all, fmt.Sprintf("%d %s", count, actionentity))
	}
	sort.Strings(all)

	max := 6
	for i := 0; i < len(all)/max; i++ {
		out = append(out, strings.Join(all[i:i+max], ", "))
	}
	if remain := len(all) % max; remain > 0 {
		out = append(out, strings.Join(all[len(all)-remain:], ", "))
	}
	return
}
