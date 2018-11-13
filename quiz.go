package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

type quizMetrics struct {
	visitCounter    *prometheus.CounterVec
	answerCounter   *prometheus.CounterVec
	answerHistogram *prometheus.HistogramVec
}

type questions struct {
	question string
	options  []string
	answer   int
	image    string
}

func getQuestions() []questions {
	return []questions{
		{
			question: "Who is older?",
			options:  []string{"Rod", "Todd"},
			answer:   0,
			image:    "rod_todd.jpg",
		},
		{
			question: "What is The Simpson's real address?",
			options:  []string{"123 Fake street", "742 Evergreen Terrace", "1024 Evergreen Terrace"},
			answer:   1,
			image:    "street.png",
		},
		{
			question: "What is the name of the startup that Homer founded and Bill Gates \"bougth\"?",
			options:  []string{"Global-Compu-Hyper-Mega-Net", "Compu-Global-Hyper-Mega-Net", "Hyper-Compu-Global-Mega-Net"},
			answer:   1,
			image:    "bill.jpg",
		},
		{
			question: "Which one is not an alias that Homer ever used?",
			options:  []string{"Max Power", "Elvis Jagger Abdul - Jabbar", "Angry Dad (Papá Rabioso)", "Rock Strongo (Fornido rock)", "Mr. Plow", "Mr. X", "Happy Dude", "Colonel Homer", "Homer Thompson", "Brian McGee", "None"},
			answer:   10,
			image:    "aliases.jpg",
		},
		{
			question: "Hank Scorpio (best super-villian ever) asks Homer about his less favorite country, and he offers two choices:",
			options:  []string{"France and Italy", "Spain and Italy", "France and Canada"},
			answer:   0,
			image:    "hank.jpg",
		},
		{
			question: "Which is the fake identity that Krusty tried to adopt when he faked his own death?",
			options:  []string{"Steve Barnes", "Mr. Snrub", "Rory B. Bellows"},
			answer:   2,
			image:    "krusty.jpg",
		},
	}
}

func msgForScore(score int, failures []string) string {
	if score == len(getQuestions()) {
		return `
  <h3>Perfect!</h3>
  <iframe src="https://giphy.com/embed/l2JdTAyoFqDY6nEis" width="480" height="366" frameBorder="0" class="giphy-embed" allowFullScreen></iframe><p><a href="https://giphy.com/gifs/season-11-the-simpsons-11x6-l2JdTAyoFqDY6nEis">via GIPHY</a></p>
`
	} else if score == len(getQuestions())-1 {
		return fmt.Sprintf(`
  <h3>meh</h3>
  <p>You failed in the question number %v</p>
  <iframe src="https://giphy.com/embed/RJSrDl3tgfKUmSfybz" width="480" height="269" frameBorder="0" class="giphy-embed" allowFullScreen></iframe><p><a href="https://giphy.com/gifs/RJSrDl3tgfKUmSfybz">via GIPHY</a></p>
`, strings.Join(failures, ", "))
	} else if score > 0 {
		return fmt.Sprintf(`
  <h3>really?</h3>
  <p>You need to improve. Have a look at these questions and try again: %v.</p>
  <iframe src="https://giphy.com/embed/k5nFcak3DT8iI" width="480" height="349" frameBorder="0" class="giphy-embed" allowFullScreen></iframe><p><a href="https://giphy.com/gifs/the-simpsons-homer-simpson-mr-burns-k5nFcak3DT8iI">via GIPHY</a></p>
`, strings.Join(failures, ", "))
	} else {
		return `
  <h3>this is so sad...</h3>
  <p>Are you a human?</p>
  <iframe src="https://giphy.com/embed/X3LZLfNMOLdGU" width="480" height="270" frameBorder="0" class="giphy-embed" allowFullScreen></iframe><p><a href="https://giphy.com/gifs/maggie-simpson-black-and-white-the-simpsons-X3LZLfNMOLdGU">via GIPHY</a></p>
`
	}
}

func presentQuiz(w http.ResponseWriter, r *http.Request) {
	var sections []string
	for idx, q := range getQuestions() {
		var options []string
		for oIdx, q := range q.options {
			option := fmt.Sprintf(`<option value="%d">%v</option>`, oIdx, q)
			options = append(options, option)
		}
		strings.Join(options, "\n")

		section := fmt.Sprintf(`
  <p><b>%d:</b> %v</p>
  <img src="/images/%v" height="300"><br>
  <select name="q%d" form="trivia">
	%v
  </select>
`, idx+1, q.question, q.image, idx, options)
		sections = append(sections, section)
	}

	doc := fmt.Sprintf(`
<!DOCTYPE html>
<html>

<head>
  <title>The Simpsons Trivia</title>
</head>

<body>

<h1>The Simpsons Trivia</h1>
<h3>How much do you know about the best cartoon show EVER?</h3>

<form action="/answer" id="trivia">
%v
  <br>
  <br>
  <input type="submit">
</form>

</body>
</html>
`, strings.Join(sections, "\n"))

	w.Write([]byte(doc))
}

func answerQuiz(metrics *quizMetrics) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			handleError(400, fmt.Sprintf("cannot parse form %v\n", err), w)
		}

		score := 0
		failures := []string{}

		for idx, q := range getQuestions() {
			gotAnsRaw := r.Form.Get(fmt.Sprintf("q%d", idx))
			gotAns, err := strconv.Atoi(gotAnsRaw)
			if err != nil {
				handleError(400, fmt.Sprintf("cannot parse selection in form %v\n", err), w)
			}

			if gotAns == q.answer {
				score = score + 1
				metrics.answerCounter.WithLabelValues(fmt.Sprintf("%d", idx+1), "hit").Inc()
			} else {
				failures = append(failures, fmt.Sprintf("%d", idx+1))
				metrics.answerCounter.WithLabelValues(fmt.Sprintf("%d", idx+1), "miss").Inc()
			}
		}

		metrics.answerHistogram.With(prometheus.Labels{}).Observe(float64(score))

		w.Write([]byte(fmt.Sprintf(`
<!DOCTYPE html>
<html>

<head>
  <title>The Simpsons Trivia -> Results</title>
</head>

<body>

<h1>Here is your result:</h1>
<h2>%d/%d</h2>

%v

</body>
</html>
`, score, len(getQuestions()), msgForScore(score, failures))))
	})
}
