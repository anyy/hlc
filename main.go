package main

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kyokomi/emoji"
	"github.com/mattn/go-runewidth"
	_ "github.com/mattn/go-sqlite3"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
)

const (
	tblSubjects = "subjects"
	layout      = "2006-01-02"
)

var (
	dayOfTheWeek = []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	monthEmoji   = map[string]string{
		"January":   ":bamboo:",
		"February":  ":chocolate_bar:",
		"March":     ":dolls:",
		"April":     ":cherry_blossom:",
		"May":       ":flags:",
		"June":      ":frog:",
		"July":      ":tanabata_tree:",
		"August":    ":sunflower:",
		"September": ":ear_of_rice:",
		"October":   ":jack_o_lantern:",
		"November":  ":maple_leaf:",
		"December":  ":christmas_tree:",
	}
)

type Subjects struct {
	ID          int
	Name        string
	Discription string
	IsDone      int
	Date        string
}

func main() {
	app := cli.NewApp()
	app.Name = "hlc"
	app.Usage = "record happy learning life"
	app.Version = "1.0.0"

	app.Commands = []cli.Command{
		{
			Name:   "init",
			Usage:  "init database",
			Action: cmdInit,
		},
		{
			Name:    "list",
			Aliases: []string{"l"},
			Usage:   "show tasks today",
			Action:  cmdList,
		},
		{
			Name:    "add",
			Aliases: []string{"a"},
			Usage:   "add a task you learn today",
			Action:  cmdAdd,
		},
		{
			Name:      "done",
			Aliases:   []string{"d"},
			Usage:     "done a task",
			ArgsUsage: "done [no]",
			Action:    cmdDone,
		},
		{
			Name:    "cal",
			Aliases: []string{"c"},
			Usage:   "show calendar",
			Action:  cmdCal,
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func cmdInit(c *cli.Context) error {
	path := dbPath()
	if fileExists(path) {
		return errors.New("init has already been done")
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	stmt := fmt.Sprintf(`
		CREATE TABLE %s (
			id          integer NOT NULL PRIMARY KEY,
			name        text NOT NULL,
			discription text NOT NULL,
			is_done     integer NOT NULL,
			date        text NOT NULL
		);
	`, tblSubjects)
	_, err = db.Exec(stmt)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("init successfully finished")

	return nil
}

func cmdList(c *cli.Context) error {
	path := dbPath()
	if !fileExists(path) {
		return errors.New("exec init first")
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatalf("cannot open database: %v", err)
	}
	defer db.Close()

	now := timeToStr(time.Now())
	subjects, err := findByDate(now)
	if err != nil {
		log.Fatalf("failed to fetch subjects: %v", err)
	}
	records := make([][]string, 0, len(subjects))
	for _, s := range subjects {
		records = append(records, []string{decorateDone(s.IsDone), strconv.Itoa(s.ID), s.Name, s.Discription, s.Date})
	}
	if len(records) > 0 {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Done", "No", "Name", "Discription", "Date"})
		table.SetBorder(true)
		table.AppendBulk(records)
		table.Render()
	}

	return nil
}

func cmdAdd(c *cli.Context) error {
	path := dbPath()
	if !fileExists(path) {
		return errors.New("exec init first")
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatalf("cannot open database: %v", err)
	}
	defer db.Close()

	var name, discription string

	fmt.Print("Subject: ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return errors.New("canceled")
	}
	if scanner.Err() != nil {
		return scanner.Err()
	}
	name = scanner.Text()
	fmt.Print("Discription: ")
	if !scanner.Scan() {
		return errors.New("canceled")
	}
	if scanner.Err() != nil {
		return scanner.Err()
	}
	discription = scanner.Text()

	now := timeToStr(time.Now())
	stmt := "INSERT INTO subjects(name, discription, is_done, date) values(?, ?, ?, ?)"
	_, err = db.Exec(stmt, name, discription, 0, now)
	if err != nil {
		log.Fatalf("failed to add: %v", err)
	}
	fmt.Println(fmt.Sprintf("\nadded %s", name))

	return nil
}

func cmdDone(c *cli.Context) error {
	if !c.Args().Present() {
		cli.ShowCommandHelp(c, "done")
		return nil
	}

	path := dbPath()
	if !fileExists(path) {
		return errors.New("exec init first")
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatalf("cannot open database: %v", err)
	}
	defer db.Close()

	id := c.Args().First()
	now := timeToStr(time.Now())
	stmt := "UPDATE subjects SET is_done = 1 WHERE id = ? AND date = ?"
	_, err = db.Exec(stmt, id, now)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(fmt.Sprintf("\nno %s has marked as done.\ngood job!", id))

	return nil
}

func cmdCal(c *cli.Context) error {
	if !fileExists(dbPath()) {
		return errors.New("exec init first")
	}
	now := time.Now()
	beginning := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
	end := beginning.AddDate(0, 1, -1)

	subjects, err := findByDates(timeToStr(beginning), timeToStr(end))
	if err != nil {
		log.Fatalf("failed to fetch subjects: %v", err)
	}

	sm := make(map[string][]Subjects)
	for _, s := range subjects {
		sm[s.Date] = append(sm[s.Date], s)
	}

	fmt.Println(header(now))
	for _, v := range dayOfTheWeek {
		fmt.Print(v + " ")
	}
	fmt.Println()

	var yesterdayRuneWidth int
	var indent, count int
	switch beginning.Weekday() {
	case time.Sunday:
		indent = 0
		count = 6
	case time.Monday:
		indent = 4
		count = 5
	case time.Tuesday:
		indent = 8
		count = 4
	case time.Wednesday:
		indent = 12
		count = 3
	case time.Thursday:
		indent = 16
		count = 2
	case time.Friday:
		indent = 20
		count = 1
	case time.Saturday:
		indent = 24
		count = 0
	}
	fmt.Print(strings.Repeat(" ", indent))
	t := beginning
	for i := 1; i < end.Day(); i++ {
		var day string
		if subs, ok := sm[t.Format(layout)]; ok {
			var cnt int
			for _, s := range subs {
				if s.IsDone == 1 {
					cnt++
				}
			}
			day = evaluateProgress(len(subs), cnt)
		}

		if day == "" {
			day = intToStrWithSpace(i, yesterdayRuneWidth, t.Weekday() == time.Sunday)
		}
		fmt.Print(fmt.Sprintf("%s ", day))
		if count <= 0 {
			count = 6
			fmt.Println()
		} else {
			count--
		}
		yesterdayRuneWidth = runewidth.StringWidth(day)
		t = t.AddDate(0, 0, 1)
	}

	return nil
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func header(t time.Time) string {
	emo := monthEmoji[t.Month().String()]
	dec := strings.Repeat(emoji.Sprint(emo), 4)
	m := t.Month().String()[:3]
	return fmt.Sprintf("\n%[1]s%[2]s %[1]s", dec, m)
}

func dbPath() string {
	return filepath.Join(os.Getenv("HOME"), "hlc")
}

func intToStrWithSpace(n, runeWidth int, isSun bool) string {
	s := 1
	if n < 10 {
		s++
	}
	if !isSun && runeWidth >= 4 {
		s--
	}
	return strings.Repeat(" ", s) + strconv.Itoa(n)
}

func evaluateProgress(all, numOfDone int) string {
	var rate float32
	rate = float32(numOfDone) / float32(all)
	p := rate * 100
	r := " "
	if p <= 30 {
		r += emoji.Sprint(":candy:")
	} else if p <= 60 {
		r += emoji.Sprint(":cake:")
	} else {
		r += emoji.Sprint(":birthday:")
	}
	return r
}

func timeToStr(t time.Time) string {
	return t.Format(layout)
}

func findByDate(d string) ([]Subjects, error) {
	db, err := sql.Open("sqlite3", dbPath())
	if err != nil {
		return []Subjects{}, err
	}
	defer db.Close()
	stmt := fmt.Sprintf("SELECT * FROM %s WHERE date = '%s';", tblSubjects, d)
	rows, err := db.Query(stmt)
	if err != nil {
		return []Subjects{}, err
	}
	defer rows.Close()

	var subjects []Subjects
	for rows.Next() {
		var id int
		var name string
		var discription string
		var isDone int
		var date string
		rows.Scan(&id, &name, &discription, &isDone, &date)
		subjects = append(subjects, NewSubjects(id, name, discription, isDone, date))
	}

	return subjects, nil
}

func findByDates(start, end string) ([]Subjects, error) {
	db, err := sql.Open("sqlite3", dbPath())
	if err != nil {
		return []Subjects{}, err
	}
	defer db.Close()

	stmt := fmt.Sprintf("SELECT * FROM %s WHERE date between '%s' and '%s';", tblSubjects, start, end)
	rows, err := db.Query(stmt)
	if err != nil {
		return []Subjects{}, err
	}
	defer rows.Close()

	var subjects []Subjects
	for rows.Next() {
		var id int
		var name string
		var discription string
		var isDone int
		var date string
		rows.Scan(&id, &name, &discription, &isDone, &date)
		subjects = append(subjects, NewSubjects(id, name, discription, isDone, date))
	}

	return subjects, nil
}

func NewSubjects(id int, name, discription string, isDone int, date string) Subjects {
	return Subjects{
		ID:          id,
		IsDone:      isDone,
		Name:        name,
		Discription: discription,
		Date:        date,
	}
}

func decorateDone(done int) string {
	if done == 1 {
		return emoji.Sprint(":white_check_mark:")
	}
	return emoji.Sprint(":white_large_square:")
}
