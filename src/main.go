package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/now"
	"io/ioutil"
	"net/http"
	"net/url"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

var sqlString string = "gimvic:GimVicServer@/gimvic"
var oneDay time.Duration = time.Date(2015, 11, 30, 0, 0, 0, 0, time.UTC).Sub(time.Date(2015, 11, 29, 0, 0, 0, 0, time.UTC))

func main() {
	http.HandleFunc("/chooserOptions", chooserOptions)
	http.HandleFunc("/data", data)
	http.HandleFunc("/teacherData", teacherData)
	http.HandleFunc("/menuUpload", menuUpload)
	http.ListenAndServe(":8080", nil)
}

func data(w http.ResponseWriter, r *http.Request) {
	queries := parseUrl(r)
	addSubs, classes, snackType, lunchType := parseQueries(queries)
	result := DataResponse{}

	result.Days = pureScedule(classes)

	if addSubs {
		result.Days = addSubstitutions(result.Days, classes)
	}

	currentDate := getPropperStartDate()
	for i := 0; i < 5; i++ {
		result.Days[i].SnackLines = getSnack(snackType, currentDate)
		result.Days[i].LunchLines = getLunch(lunchType, currentDate)
		currentDate = currentDate.Add(oneDay)
	}

	validUntil := getPropperStartDate().Add(5 * oneDay)
	result.ValidUntil = dateToStr(validUntil)

	jsonStr, err := json.Marshal(result)
	check(err)
	fmt.Fprint(w, string(jsonStr))
}

func teacherData(w http.ResponseWriter, r *http.Request) {
	queries := parseUrl(r)
	addSubs, teacher, snackType, lunchType := parseTeacherQueries(queries)
	result := DataResponse{}

	result.Days = pureTeacherScedule(teacher)
	if addSubs {
		result.Days = addTeacherSubstitutions(result.Days, teacher)
	}

	currentDate := getPropperStartDate()
	for i := 0; i < 5; i++ {
		result.Days[i].SnackLines = getSnack(snackType, currentDate)
		result.Days[i].LunchLines = getLunch(lunchType, currentDate)
		currentDate = currentDate.Add(oneDay)
	}

	validUntil := getPropperStartDate().Add(5 * oneDay)
	result.ValidUntil = dateToStr(validUntil)

	jsonStr, err := json.Marshal(result)
	check(err)
	fmt.Fprint(w, string(jsonStr))
}

func menuUpload(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	csv := r.Form["data"][0]
	datetime := time.Now()
	fileName := "menu_files/" + dateToStr(datetime) + "-" + strconv.Itoa(datetime.Hour()) + "-" + strconv.Itoa(datetime.Minute()) + "-" + strconv.Itoa(datetime.Second()) + ".csv"
	err := ioutil.WriteFile(fileName, []byte(csv), 0777)
	check(err)

	_, err = exec.Command("gimvic-data-updater menu " + fileName).Output()
	check(err)
}

func dateToStr(date time.Time) string {
	return strconv.Itoa(date.Year()) + "-" + strconv.Itoa(int(date.Month())) + "-" + strconv.Itoa(date.Day())
}

func getSnack(typeStr string, date time.Time) []string {
	con, err := sql.Open("mysql", sqlString)
	check(err)
	defer con.Close()

	if typeStr == "navadna" {
		typeStr = "normal"
	}
	if typeStr == "vegetarijanska" {
		typeStr = "veg"
	}
	if typeStr == "vegetarijanska_s_perutnino_in_ribo" {
		typeStr = "veg_per"
	}
	if typeStr == "sadnozelenjavna" {
		typeStr = "sadnozel"
	}
	rows, err := con.Query("select " + typeStr + " from snack where date='" + dateToStr(date) + "';")
	check(err)
	var temp string
	rows.Next()
	rows.Scan(&temp)

	return strings.Split(temp, ";")
}

func getLunch(typeStr string, date time.Time) []string {
	con, err := sql.Open("mysql", sqlString)
	check(err)
	defer con.Close()

	if typeStr == "navadno" {
		typeStr = "normal"
	}
	if typeStr == "vegetarijansko" {
		typeStr = "veg"
	}
	rows, err := con.Query("select " + typeStr + " from lunch where date='" + dateToStr(date) + "';")
	check(err)
	var temp string
	rows.Next()
	rows.Scan(&temp)

	return strings.Split(temp, ";")
}

func addSubstitutions(days [5]Day, classes []string) [5]Day {
	date := getPropperStartDate()

	for i := 0; i < 5; i++ {
		where := "("
		for _, class := range classes {
			if where != "(" {
				where += " or "
			}
			where += "class='" + class + "'"
		}
		where += ") and date='" + dateToStr(date) + "'"

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
			days[i].Lessons[lesson-1].Classes = addIfNeeded(days[i].Lessons[lesson-1].Classes, class)

			days[i].Lessons[lesson-1].Teachers = days[i].Lessons[lesson-1].Teachers[:0]
			days[i].Lessons[lesson-1].Teachers = addIfNeeded(days[i].Lessons[lesson-1].Teachers, teacher)

			days[i].Lessons[lesson-1].Subjects = days[i].Lessons[lesson-1].Subjects[:0]
			days[i].Lessons[lesson-1].Subjects = addIfNeeded(days[i].Lessons[lesson-1].Subjects, subject)

			days[i].Lessons[lesson-1].Classrooms = days[i].Lessons[lesson-1].Classrooms[:0]
			days[i].Lessons[lesson-1].Classrooms = addIfNeeded(days[i].Lessons[lesson-1].Classrooms, classroom)

			days[i].Lessons[lesson-1].Note = note
		}

		date = date.Add(oneDay)
	}

	return days
}

func addTeacherSubstitutions(days [5]Day, teacher string) [5]Day {
	date := getPropperStartDate()

	for i := 0; i < 5; i++ {
		where := "teacher='" + teacher + "' or absent_teacher='" + teacher + "' and date='" + dateToStr(date) + "'"

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
			days[i].Lessons[lesson-1].Classes = addIfNeeded(days[i].Lessons[lesson-1].Classes, class)

			days[i].Lessons[lesson-1].Teachers = days[i].Lessons[lesson-1].Teachers[:0]
			days[i].Lessons[lesson-1].Teachers = addIfNeeded(days[i].Lessons[lesson-1].Teachers, teacher)

			days[i].Lessons[lesson-1].Subjects = days[i].Lessons[lesson-1].Subjects[:0]
			days[i].Lessons[lesson-1].Subjects = addIfNeeded(days[i].Lessons[lesson-1].Subjects, subject)

			days[i].Lessons[lesson-1].Classrooms = days[i].Lessons[lesson-1].Classrooms[:0]
			days[i].Lessons[lesson-1].Classrooms = addIfNeeded(days[i].Lessons[lesson-1].Classrooms, classroom)

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
	resultClasses := q["classes[]"]
	resSnackType := q["snackType"][0]
	resLunchType := q["lunchType"][0]

	return addSubs, resultClasses, resSnackType, resLunchType

}

func parseTeacherQueries(q map[string][]string) (addSubstitutions bool, teacher string, snackType, lunchType string) {
	addSubs := false
	if q["addSubstitutions"][0] == "true" {
		addSubs = true
	}
	resultTeacher := q["teacher"][0]
	resSnackType := q["snackType"][0]
	resLunchType := q["lunchType"][0]

	return addSubs, resultTeacher, resSnackType, resLunchType

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
		days[day-1].Lessons[lesson-1].Classes = addIfNeeded(days[day-1].Lessons[lesson-1].Classes, class)
		days[day-1].Lessons[lesson-1].Teachers = addIfNeeded(days[day-1].Lessons[lesson-1].Teachers, teacher)
		days[day-1].Lessons[lesson-1].Subjects = addIfNeeded(days[day-1].Lessons[lesson-1].Subjects, subject)
		days[day-1].Lessons[lesson-1].Classrooms = addIfNeeded(days[day-1].Lessons[lesson-1].Classrooms, classroom)
	}

	return days
}

func pureTeacherScedule(teacher string) [5]Day {
	var days [5]Day
	con, err := sql.Open("mysql", sqlString)
	check(err)
	defer con.Close()

	rows, err := con.Query("select class, subject, classroom, day, lesson from schedule where teacher='" + teacher + "';")
	check(err)
	var class, subject, classroom string
	var day, lesson int
	for rows.Next() {
		rows.Scan(&class, &subject, &classroom, &day, &lesson)
		days[day-1].Lessons[lesson-1].Classes = addIfNeeded(days[day-1].Lessons[lesson-1].Classes, class)
		days[day-1].Lessons[lesson-1].Teachers = addIfNeeded(days[day-1].Lessons[lesson-1].Teachers, teacher)
		days[day-1].Lessons[lesson-1].Subjects = addIfNeeded(days[day-1].Lessons[lesson-1].Subjects, subject)
		days[day-1].Lessons[lesson-1].Classrooms = addIfNeeded(days[day-1].Lessons[lesson-1].Classrooms, classroom)
	}

	return days
}

func addIfNeeded(original []string, add string) []string {
	for _, item := range original {
		if item == add {
			return original
		}
	}
	return append(original, add)
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

	//fill additional teachers
	rows, err = con.Query("select teacher from teachers;")
	check(err)
	for rows.Next() {
		var temp string
		rows.Scan(&temp)
		response.Teachers = append(response.Teachers, temp)
	}

	sort.Strings(response.MainClasses)
	sort.Strings(response.AdditionalClasses)
	sort.Strings(response.Teachers)

	response.LunchTypes = [2]string{"navadno", "vegetarijansko"}
	response.SnackTypes = [4]string{"navadna", "vegetarijanska", "vegetarijanska_s_perutnino_in_ribo", "sadnozelenjavna"}
	responseStr, err := json.Marshal(response)
	check(err)
	fmt.Fprint(w, string(responseStr))
}

func getPropperStartDate() time.Time {
	now.FirstDayMonday = true
	result := now.BeginningOfWeek()
	current := time.Now()
	if current.Weekday() == time.Saturday || current.Weekday() == time.Sunday {
		result = result.Add(7 * oneDay)
	}
	return result
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
	MainClasses       []string  `json:"mainClasses,omitempty"`
	AdditionalClasses []string  `json:"additionalClasses,omitempty"`
	Teachers          []string  `json:"teachers,omitempty"`
	SnackTypes        [4]string `json:"snackTypes,omitempty"`
	LunchTypes        [2]string `json:"lunchTypes,omitempty"`
}

type DataResponse struct {
	Days       [5]Day `json:"days,omitempty"`
	ValidUntil string `json:"validUntil,omitempty"`
}

type Day struct {
	Lessons    [8]Lesson `json:"lessons,omitempty"`
	SnackLines []string  `json:"snackLines,omitempty"`
	LunchLines []string  `json:"lunchLines,omitempty"`
	SnackType  string    `json:"snackType,omitempty"`
	LunchType  string    `json:"lunchType,omitempty"`
}

type Lesson struct {
	Subjects       []string `json:"subjects,omitempty"`
	Teachers       []string `json:"teachers,omitempty"`
	Classrooms     []string `json:"classrooms,omitempty"`
	Classes        []string `json:"classes,omitempty"`
	Note           string   `json:"note,omitempty"`
	Lesson         int      `json:"lesson,omitempty"`
	Day            int      `json:"day,omitempty"`
	IsSubstitution bool     `json:"substitution,omitempty"`
}
