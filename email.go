package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// TODO: change to interface so we can unit test

// ErrSubjectFmt ...
var ErrSubjectFmt = "CountMyReps was unable to parse your subject. Please provide FOUR comma separated numbers like: `5, 10, 15, 20` where the numbers represent pull ups, push ups, squats, and situps respectively. You provided \"%s\""

// ErrToAddrFmt ...
var ErrToAddrFmt = "CountMyReps only accepts emails to " + NewEmail + ", you sent to \"%s\""

// ErrFromFmt ...
var ErrFromFmt = "CountMyReps only accepts mail from the sendgrid domain. You used \"%s\""

// ErrUnexpectedFmt ...
var ErrUnexpectedFmt = "CountMyReps experienced an unexpected error, please try again later. Error: %s"

// SendErrorEmail sets up the error message and then calls sendEmail
func SendErrorEmail(rcpt string, originalAddressTo string, subject string, msg string) error {
	officeList := strings.Join(Offices, ", ")
	msgFmt := `
	<h3>Uh oh!</h3>
	<p>
	There was an error with your CountMyReps Submission.<br /><br />
    Make sure that you addressed your email to %s<br />
    Make sure that your subject line was FOUR comma separated numbers, like: 5, 10, 15, 20<br />
    If you were trying to set your office location, make sure you choose one from:<br />
	%s<br />
	(This should be sent in its own email).
    </p>
	<p>
    Details from received message:<br />
    Addessed to: %s<br />
    Subject: %s<br />
    Time: %s<br />
	Error: %s<br />
	</p>`
	return sendEmail(rcpt, "Error with your submission", fmt.Sprintf(msgFmt, NewEmail, officeList, originalAddressTo, subject, time.Now().String(), msg))
}

func officeComparisonUpdate(userOffice string, officeStats map[string]Stats) string {
	var leadOffice string
	var currentLeadCount int
	for office, stats := range officeStats {
		if stats.RepsPerPersonParticipatingPerDay >= currentLeadCount {
			leadOffice = office
		}
	}
	var msg string
	officePercent := fmt.Sprintf("%d%%", officeStats[userOffice].PercentParticipating)
	officePerDay := officeStats[userOffice].RepsPerPersonParticipatingPerDay
	if userOffice == leadOffice {
		msg = fmt.Sprintf("Your office is leading with %s%% participating, with those Gridders doing %d reps per day!", officePercent, officePerDay)
	} else {
		msg = fmt.Sprintf("Your office has %s%% participating, with those Gridders doing %d reps per day. With a little effort, you can catch up to the %s office who have %d%% particpating, doing %d reps per day.",
			officePercent, officePerDay,
			leadOffice,
			officeStats[leadOffice].PercentParticipating, officeStats[leadOffice].RepsPerPersonParticipatingPerDay)
	}
	return msg
}

// SendSuccessEmail sets up the success message and calls sendEmail
func SendSuccessEmail(to string) error {
	office := getUserOffice(to)
	officeStats := getOfficeStats()
	var officeMsg string
	var forTheTeam string
	if office == "" || office == "Unknown" {
		officeMsg = fmt.Sprintf("You've not linked your reps to an office. Send an email to %s with your office in the subject line. Valid office choices are: <br />%s", NewEmail, strings.Join(Offices, ", "))
		forTheTeam = ""
	} else {
		officeMsg = officeComparisonUpdate(office, officeStats)
		forTheTeam = fmt.Sprintf(" for the %s team", office)
	}
	total := totalReps(getUserReps(to))
	days := int(time.Since(StartDate).Hours() / float64(24))
	if days == 0 {
		days = 1 // avoid divide by zero
	}
	avg := total / days

	var data []string
	for officeName, stats := range officeStats {
		data = append(data, fmt.Sprintf("%s: %d", officeName, stats.TotalReps))
	}

	officeTotals := "The office totals are: " + strings.Join(data, ", ")

	msg := fmt.Sprintf(`<h3>Keep it up!</h3>
	<p>
	You've logged a total of %d%s, an average of %d per day.
	</p>
	<p>
	%s
	</p>
	<p>
	%s
	</p>`, total, forTheTeam, avg, officeMsg, officeTotals)

	return sendEmail(to, "Success!", fmt.Sprintf(msg))
}

func sendEmail(to string, subject string, msg string) error {
	from := mail.NewEmail("CountMyReps", "automailer@countmyreps.com")
	// at this point, all recipients _should_ be firstname.lastname@sendgrid.com or firstname@sendgrid.com
	toName := strings.Split(to, ".")[0]
	if strings.Contains(toName, "@") {
		toName = strings.Split(toName, "@")[0]
	}
	toAddr := mail.NewEmail(toName, to)

	msg = `<img src="http://countmyreps.com/images/mustache-thin.jpg" style="margin:auto; width:300px; display:block"/>` + msg

	content := mail.NewContent("text/html", msg)
	m := mail.NewV3MailInit(from, subject, toAddr, content)

	request := sendgrid.GetRequest(os.Getenv("SENDGRID_API_KEY"), "/v3/mail/send", "https://api.sendgrid.com")
	request.Method = "POST"
	request.Body = mail.GetRequestBody(m)
	response, err := sendgrid.API(request)
	if err != nil {
		return err
	}
	if !(response.StatusCode == http.StatusOK || response.StatusCode == http.StatusAccepted) {
		return fmt.Errorf("unexpected status code from SendGrid: %d - %q", response.StatusCode, response.Body)
	}
	return nil
}

// extractEmailAddr gets the email address from the email string
// John <Smith@example.com>
// <Smith@example.com>
// smith@example.com
// ^^ all gitve smith@example.com
func extractEmailAddr(email string) string {
	if !strings.Contains(email, "<") {
		return email
	}
	var extracted []rune
	var capture bool
	for _, r := range email {
		if string(r) == "<" {
			capture = true
			continue
		}
		if string(r) == ">" {
			capture = false
			continue
		}
		if capture {
			extracted = append(extracted, r)
		}
	}
	return string(extracted)
}
