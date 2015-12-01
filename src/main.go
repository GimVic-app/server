package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/now"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var sqlString string = "gimvic:GimVicServer@/gimvic"
var oneDay time.Duration = time.Date(2015, 11, 30, 0, 0, 0, 0, time.UTC).Sub(time.Date(2015, 11, 29, 0, 0, 0, 0, time.UTC))

func main() {
	http.HandleFunc("/chooserOptions", chooserOptions)
	http.HandleFunc("/data", data)
	http.ListenAndServe(":8080", nil)
}

func data(w http.ResponseWriter, r *http.Request) {
	queries := parseUrl(r)
	addSubs, classes, _, _ := parseQueries(queries)
	result := DataResponse{}

	result.Days = pureScedule(classes)

	if addSubs {
		result.Days = addSubstitutions(result.Days, classes)
	}
	jsonStr, err := json.Marshal(result)
	check(err)
	fmt.Fprint(w, string(jsonStr))
}

func addSubstitutions(days [5]Day, classes []string) [5]Day {
	now.FirstDayMonday = true
	date := now.BeginningOfWeek()

	for i := 0; i < 5; i++ {
		where := "("
		for _, class := range classes {
			if where != "(" {
				where += " or "
			}
			where += "class='" + class + "'"
		}
		dateStr := strconv.Itoa(date.Year()) + "-" + strconv.Itoa(int(date.Month())) + "-" + strconv.Itoa(date.Day())
		where += ") and date='" + dateStr + "'"

		con, err := sql.Open("mysql", sqlString)
		check(err)
		defer con.Close()
		rows, err := con.Query("select class, teacher, subject, classroom, lesson, note from substitutions where " + where + ";")
		check(err)
		var class, teacher, subject, classroom, note string
		var lesson int
		for rows.Next() {
			rows.Scan(&class, &teacher, &subject, &classroom, &lesson, &note)

			days[i].Lessons[lesson-1].IsSubstitution = true

			days[i].Lessons[lesson-1].Classes = days[i].Lessons[lesson-1].Classes[:0]
			days[i].Lessons[lesson-1].Classes = append(days[i].Lessons[lesson-1].Classes, class)

			days[i].Lessons[lesson-1].Teachers = days[i].Lessons[lesson-1].Teachers[:0]
			days[i].Lessons[lesson-1].Teachers = append(days[i].Lessons[lesson-1].Teachers, teacher)

			days[i].Lessons[lesson-1].Subjects = days[i].Lessons[lesson-1].Subjects[:0]
			days[i].Lessons[lesson-1].Subjects = append(days[i].Lessons[lesson-1].Subjects, subject)

			days[i].Lessons[lesson-1].Classrooms = days[i].Lessons[lesson-1].Classrooms[:0]
			days[i].Lessons[lesson-1].Classrooms = append(days[i].Lessons[lesson-1].Classrooms, classroom)

			days[i].Lessons[lesson-1].Note = note
		}

		date = date.Add(oneDay)
	}

	return days
}

func parseQueries(q map[string][]string) (addSubstitutions bool, classes []string, snackType, lunchType string) {
	addSubs := false
	if q["addSubstitutions"][0] == "true" {
		addSubs = true
	}
	resultClasses := q["classes"]
	resSnackType := q["snackType"][0]
	resLunchType := q["lunchType"][0]

	return addSubs, resultClasses, resSnackType, resLunchType

}

func pureScedule(classes []string) [5]Day {
	var days [5]Day
	con, err := sql.Open("mysql", sqlString)
	check(err)
	defer con.Close()

	where := ""
	for _, class := range classes {
		if where != "" {
			where += " or "
		}
		where += "class='" + class + "'"
	}
	rows, err := con.Query("select class, teacher, subject, classroom, day, lesson from schedule where " + where + ";")
	check(err)
	var class, teacher, subject, classroom string
	var day, lesson int
	for rows.Next() {
		rows.Scan(&class, &teacher, &subject, &classroom, &day, &lesson)
		days[day-1].Lessons[lesson-1].Classes = append(days[day-1].Lessons[lesson-1].Classes, class)
		days[day-1].Lessons[lesson-1].Teachers = append(days[day-1].Lessons[lesson-1].Teachers, teacher)
		days[day-1].Lessons[lesson-1].Subjects = append(days[day-1].Lessons[lesson-1].Subjects, subject)
		days[day-1].Lessons[lesson-1].Classrooms = append(days[day-1].Lessons[lesson-1].Classrooms, classroom)
	}

	return days
}
func chooserOptions(w http.ResponseWriter, r *http.Request) {
	response := ChooserOptionsResponse{}
	con, err := sql.Open("mysql", sqlString)
	check(err)
	defer con.Close()

	//fill main classes
	rows, err := con.Query("select class from classes where main=1;")
	check(err)
	for rows.Next() {
		var temp string
		rows.Scan(&temp)
		response.MainClasses = append(response.MainClasses, temp)
	}

	//fill additional classes
	rows, err = con.Query("select class from classes where main=0;")
	check(err)
	for rows.Next() {
		var temp string
		rows.Scan(&temp)
		response.AdditionalClasses = append(response.AdditionalClasses, temp)
	}

	responseStr, err := json.Marshal(response)
	check(err)
	fmt.Fprint(w, string(responseStr))
}

func parseUrl(r *http.Request) map[string][]string {
	str := r.URL.String()
	u, err := url.Parse(str)
	check(err)
	m, err := url.ParseQuery(u.RawQuery)
	check(err)
	return m
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

type ChooserOptionsResponse struct {
	MainClasses       []string `json:"mainClasses,omitempty"`
	AdditionalClasses []string `json:"additionalClasses,omitempty"`
	ValidUntil        string   `json:"validUntil,omitempty"`
}

type DataResponse struct {
	Days [5]Day `json:"days,omitempty"`
	Hash string `json:"hash,omitempty"`
}

type Day struct {
	Lessons    [8]Lesson `json:"lessons,omitempty"`
	SnackLines []string  `json:"snackLines,omitempty"`
	LunchLines []string  `json:"lunchLines,omitempty"`
}

type Lesson struct {
	Subjects       []string `json:"subjects,omitempty"`
	Teachers       []string `json:"teachers,omitempty"`
	Classrooms     []string `json:"classrooms,omitempty"`
	Classes        []string `json:"classes,omitempty"`
	Note           string   `json:"note,omitempty"`
	Lesson         int      `json:"lesson,omitempty"`
	Day            int      `json:"day,omitempty"`
	IsSubstitution bool     `json:"substitution"`
}
