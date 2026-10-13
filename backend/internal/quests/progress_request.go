package quests

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const MaxProgressBodyBytes = 64 * 1024

type progressRequest struct {
	ExerciseID      string
	DurationSeconds int64
	Submission      ProgressSubmission
}

// decodeProgressRequest uses tokens so duplicate keys cannot be silently lost
// by encoding/json's usual last-key-wins map/struct decoding.
func decodeProgressRequest(r io.Reader) (progressRequest, error) {
	var req progressRequest
	data, err := io.ReadAll(io.LimitReader(r, MaxProgressBodyBytes+1))
	if err != nil || len(data) > MaxProgressBodyBytes {
		return req, ErrInvalidAnswers
	}
	d := json.NewDecoder(strings.NewReader(string(data)))
	d.UseNumber()
	tok, err := d.Token()
	if err != nil || tok != json.Delim('{') {
		return req, ErrInvalidAnswers
	}
	seen := map[string]bool{}
	for d.More() {
		tok, err = d.Token()
		if err != nil {
			return req, err
		}
		key, ok := tok.(string)
		if !ok || seen[key] {
			return req, ErrInvalidAnswers
		}
		seen[key] = true
		switch key {
		case "exercise_id":
			v, e := d.Token()
			if e != nil {
				return req, e
			}
			s, ok := v.(string)
			if !ok || strings.TrimSpace(s) == "" {
				return req, ErrInvalidAnswers
			}
			req.ExerciseID = s
		case "duration_seconds":
			v, e := d.Token()
			if e != nil {
				return req, e
			}
			n, ok := v.(json.Number)
			if !ok {
				return req, ErrInvalidAnswers
			}
			req.DurationSeconds, err = n.Int64()
			if err != nil {
				return req, err
			}
		case "content_revision":
			v, e := d.Token()
			if e != nil {
				return req, e
			}
			s, ok := v.(string)
			if !ok || s == "" {
				return req, ErrInvalidAnswers
			}
			req.Submission.Revision = &s
		case "user_answers":
			tok, err = d.Token()
			if err != nil || tok != json.Delim('{') {
				return req, ErrInvalidAnswers
			}
			req.Submission.Answers = map[string]string{}
			for d.More() {
				v, e := d.Token()
				if e != nil {
					return req, e
				}
				id, ok := v.(string)
				if !ok {
					return req, ErrInvalidAnswers
				}
				if _, exists := req.Submission.Answers[id]; exists {
					return req, ErrInvalidAnswers
				}
				v, e = d.Token()
				if e != nil {
					return req, e
				}
				choice, ok := v.(string)
				if !ok {
					return req, ErrInvalidAnswers
				}
				req.Submission.Answers[id] = choice
			}
			tok, err = d.Token()
			if err != nil || tok != json.Delim('}') {
				return req, ErrInvalidAnswers
			}
		default:
			return req, fmt.Errorf("%w: unknown field", ErrInvalidAnswers)
		}
	}
	tok, err = d.Token()
	if err != nil || tok != json.Delim('}') {
		return req, ErrInvalidAnswers
	}
	if _, err = d.Token(); err != io.EOF {
		return req, ErrInvalidAnswers
	}
	if !seen["exercise_id"] || !seen["duration_seconds"] {
		return req, ErrInvalidAnswers
	}
	return req, nil
}
