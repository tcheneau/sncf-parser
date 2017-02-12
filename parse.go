package main

import "fmt"
import "golang.org/x/net/html"
import "os"
import "log"
import "time"
import "strings"

type travel struct {
	Start     string
	End       string
	Duration  string
	From      string
	To        string
	Car       string
	Seat      string
	Train     string
	Ref       string
	PlaceType string
}

func (t travel) String() string {
	return fmt.Sprintf("%s - %s (%s - %s) - v:%s pl:%s", t.Start, t.End, t.Duration, t.Train, t.Car, t.Seat)
}

type tokType int

const (
	startTok tokType = iota
	endTok
	durationTok
	fromTok
	toTok
	seatTok
	ignoreTok
	trainTok
	refTok
	placetypeTok
	// token for states
	departureTok
	arrivalTok
)

func parseentry(t *html.Tokenizer) (travel, error) {
	var nesting uint = 1
	var entry travel
	var last tokType
	var nestedTok tokType

	for {
		tt := t.Next()
		if tt == html.ErrorToken {
			return entry, t.Err()
		}

		switch tt {
		case html.StartTagToken:
			tok := t.Token()
			if tok.Data == "div" || tok.Data == "span" {
				nesting++
				for _, a := range tok.Attr {
					if a.Key == "class" {
						switch a.Val {
						case "departure":
							nestedTok = departureTok
							last = ignoreTok
						case "arrival":
							nestedTok = arrivalTok
							last = ignoreTok
						case "travelTime libStatus4":
							switch nestedTok {
							case departureTok:
								last = startTok
							case arrivalTok:
								last = endTok
							default:
								panic("state machine should not end up here")
							}
						case "travelStation":
							switch nestedTok {
							case departureTok:
								last = fromTok
							case arrivalTok:
								last = toTok
							default:
								panic("state machine should not end up here")
							}
						case "duration":
							last = durationTok
						case "placementInfo":
							nestedTok = seatTok
						case "trainInfo":
							last = trainTok
						case "placementType":
							last = placetypeTok
						case "prnLocatorValue":
							last = refTok
						case "":
							if nestedTok == seatTok {
								last = seatTok
							} else {
								last = ignoreTok
							}
						default:
							last = ignoreTok
						}
					}
				}
			}
		case html.TextToken:
			tok := t.Token()
			text := strings.TrimSpace(tok.String())
			if len(text) == 0 {
				continue
			}

			switch last {
			case ignoreTok:
				continue
			case startTok:
				entry.Start = text
			case endTok:
				entry.End = text
			case durationTok:
				entry.Duration = text[len(text)-5:]
			case fromTok:
				entry.From = text
			case toTok:
				entry.To = text
			case seatTok:
				if strings.Contains(text, "Voiture") {
					entry.Car = text[len(text)-3:]
				}
				if strings.Contains(text, "Place") {
					entry.Seat = text[len(text)-3:]
				}
			case trainTok:
				entry.Train = text[len(text)-4:]
			case refTok:
				entry.Ref = text
			case placetypeTok:
				entry.PlaceType = text
			default:
				panic("Token not supported")
			}
		case html.EndTagToken:
			tok := t.Token()
			if tok.Data == "div" || tok.Data == "span" {
				nesting--
				if nesting == 0 {
					return entry, nil
				}
			}
		}
	}
}

func main() {
	var day string
	if len(os.Args) != 2 {
		os.Exit(1)
	}
	f, err := os.Open(os.Args[1])

	if err != nil {
		log.Fatal(err)
	}

	t := html.NewTokenizer(f)

	for {
		// class="bookBlockInput"
		// class="travelTime libStatus4"
		tt := t.Next()
		if tt == html.ErrorToken {
			return
		}
		switch tt {
		case html.StartTagToken:
			tok := t.Token()
			if tok.Data == "div" {
				for _, a := range tok.Attr {
					if a.Key == "id" && strings.HasPrefix(a.Val, "daysubblock_") {
						day = strings.Split(a.Val, "_")[1]
						date, err := time.Parse("02/01/2006", day)
						if err != nil {
							panic("date parse error")
						}
						day = date.Format("Mon 02/01")
					}
					if a.Key == "class" && a.Val == "bookingBlockContent bookingBlockStatus4" {
						entry, err := parseentry(t)
						if err == nil {
							fmt.Printf("%+v - %s\n", entry, day)
						}
					}
				}

			}
		}
	}
}
