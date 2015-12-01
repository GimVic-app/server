package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
	"net/url"
)

var sqlString string = "gimvic:GimVicServer@/gimvic"

func main() {
	http.HandleFunc("/chooserOptions", chooserOptions)
	http.HandleFunc("/data", data)
	http.ListenAndServe(":8080", nil)
}

func data(w http.ResponseWriter, r *http.Request) {
	queries := parseUrl(r)

	result := DataResponse{}
	if queries["type"][0] == "hybrid" {
		result.Days = pureScedule(queries)
	}

	jsonStr, err := json.Marshal(result)
	check(err)
	fmt.Fprint(w, string(jsonStr))
}

func pureScedule(queries map[string][]string) [5]Day {
	var days [5]Day
	con, err := sql.Open("mysql", sqlString)
	check(err)
	defer con.Close()

	where := ""
	for _, class := range queries["classes"] {
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
		fmt.Println(lesson - 1)
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
