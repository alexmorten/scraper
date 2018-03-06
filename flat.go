package scraper

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/yhat/scrape"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

//ROOTURL is the url of the scrape
var ROOTURL = "https://www.wg-gesucht.de/"

//Flat is the representation of a flat
type Flat struct {
	Price float64
	Area  float64
	Title string
	URL   string
}

//ParentNodeMatcher matches a parent flat node
func ParentNodeMatcher(n *html.Node) bool {
	class := scrape.Attr(n, "class")
	if strings.Contains(class, "list-details-ad-border") && !strings.Contains(class, "panel-hidden") {
		return len(scrape.FindAll(n, priceNodeMatcher)) == 1 && len(scrape.FindAll(n, titleNodeMatcher)) == 1
	}
	return false
}

func priceNodeMatcher(n *html.Node) bool {
	if n.DataAtom == atom.A {
		return scrape.Attr(n, "class") == "detailansicht" && strings.Contains(scrape.Text(n), "€")
	}
	return false
}

func titleNodeMatcher(n *html.Node) bool {
	if n.Parent == nil {
		return false
	}

	if n.DataAtom == atom.A {
		return (scrape.Attr(n, "class") == "detailansicht" &&
			!strings.Contains(scrape.Text(n), "€") &&
			strings.Contains(scrape.Attr(n.Parent, "class"), "headline-list-view") &&
			strings.Contains(scrape.Attr(n.Parent, "class"), "noprint"))
	}
	return false
}

// FindFlats in the node
func FindFlats(n *html.Node) []*Flat {
	flatNodes := scrape.FindAll(n, ParentNodeMatcher)
	flats := []*Flat{}
	for _, node := range flatNodes {
		flat, err := NewFlat(node)
		if err != nil {
			fmt.Println(err)
		} else {
			flats = append(flats, flat)
		}
	}
	return flats
}

//NewFlat constructs a Flat from a parent node
func NewFlat(n *html.Node) (*Flat, error) {
	flat := &Flat{}

	priceNode, ok := scrape.Find(n, priceNodeMatcher)
	if !ok {
		return nil, errors.New("no priceNode found")
	}
	titleNode, ok := scrape.Find(n, titleNodeMatcher)
	if !ok {
		return nil, errors.New("no titleNode found")
	}
	price, err := parsePrice(priceNode)
	if err != nil {
		return nil, err
	}
	flat.Price = price

	area, err := parseArea(priceNode)
	if err != nil {
		return nil, err
	}
	flat.Area = area

	flat.Title = scrape.Text(titleNode)
	flat.URL = parseURL(titleNode)

	return flat, nil
}

func parsePrice(n *html.Node) (float64, error) {
	exp := regexp.MustCompile("(?P<price>\\d*)€")
	priceString := exp.FindStringSubmatch(scrape.Text(n))[1]
	return strconv.ParseFloat(priceString, 64)
}

func parseArea(n *html.Node) (float64, error) {
	exp := regexp.MustCompile("(?P<area>\\d*)m")
	areaString := exp.FindStringSubmatch(scrape.Text(n))[1]
	return strconv.ParseFloat(areaString, 64)
}

func parseURL(n *html.Node) string {
	s := scrape.Attr(n, "href")
	if s == "" {
		return s
	}
	if !strings.Contains(s, "http") {
		s = ROOTURL + s
	}
	return s
}

// By is the type of a "less" function that defines the ordering of its Planet arguments.
type By func(f1, f2 *Flat) bool

// Sort is a method on the function type, By, that sorts the argument slice according to the function.
func (by By) Sort(flats []*Flat) {
	ps := &flatSorter{
		flats: flats,
		by:    by, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Sort(ps)
}

// planetSorter joins a By function and a slice of Planets to be sorted.
type flatSorter struct {
	flats []*Flat
	by    func(p1, p2 *Flat) bool // Closure used in the Less method.
}

// Len is part of sort.Interface.
func (s *flatSorter) Len() int {
	return len(s.flats)
}

// Swap is part of sort.Interface.
func (s *flatSorter) Swap(i, j int) {
	s.flats[i], s.flats[j] = s.flats[j], s.flats[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (s *flatSorter) Less(i, j int) bool {
	return s.by(s.flats[i], s.flats[j])
}
