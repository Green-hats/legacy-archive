package mikan

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"

	"ani-rss/internal/config"
	"ani-rss/internal/groupregex"
	"ani-rss/internal/model"
	"ani-rss/internal/util"
)

// Host returns the configured mikan host.
func Host() string {
	h := config.Get().MikanHost
	if strings.TrimSpace(h) == "" {
		h = "https://mikanani.me"
	}
	return strings.TrimSuffix(h, "/")
}

func fetch(url string) (*html.Node, error) {
	b, err := util.GetBytes(url)
	if err != nil {
		return nil, err
	}
	doc, err := html.Parse(strings.NewReader(string(b)))
	if err != nil {
		return nil, err
	}
	return doc, nil
}

// Search returns the mikan listing for a query or season.
func Search(text string, season *model.MikanSeason) *model.Mikan {
	mikan := &model.Mikan{Seasons: []model.MikanSeason{}, Weeks: []model.MikanWeek{}}
	url := Host()
	if strings.TrimSpace(text) != "" {
		if m := regexp.MustCompile(`^id: (\d+)$`).FindStringSubmatch(text); len(m) > 1 {
			info := GetMikanInfo(m[1])
			mikan.Weeks = append(mikan.Weeks, model.MikanWeek{
				WeekLabel: "Search",
				Items:     []model.MikanInfo{*info},
			})
			mikan.TotalItem = 1
			return mikan
		}
		url = url + "/Home/Search?searchstr=" + strings.ReplaceAll(strings.TrimSpace(text), " ", "%20")
	} else if season != nil && season.Year > 0 && season.Season != "" {
		url = url + "/Home/BangumiCoverFlowByDayOfWeek?year=" + strconv.Itoa(season.Year) + "&seasonStr=" + season.Season
	}

	doc, err := fetch(url)
	if err != nil {
		return mikan
	}

	// seasons dropdown
	for _, dateSelect := range findByClass(doc, "date-select") {
		dateText := ""
		if el := firstByClass(dateSelect, "date-text"); el != nil {
			dateText = strings.TrimSpace(nodeText(el))
		}
		for _, dd := range findByClass(dateSelect, "dropdown-menu") {
			collectSeasons(dd, dateText, &mikan.Seasons)
		}
	}

	skBangumis := findByClass(doc, "sk-bangumi")
	if len(skBangumis) == 0 {
		anUl := firstByClass(doc, "an-ul")
		infos := collectAni(anUl)
		mikan.Weeks = append(mikan.Weeks, model.MikanWeek{WeekLabel: "Search", Items: infos})
	} else {
		for _, sk := range skBangumis {
			infos := collectAni(sk)
			if len(infos) == 0 {
				continue
			}
			label := ""
			if sk.FirstChild != nil {
				label = strings.TrimSpace(nodeText(sk.FirstChild))
			}
			mikan.Weeks = append(mikan.Weeks, model.MikanWeek{WeekLabel: label, Items: infos})
		}
	}
	for _, w := range mikan.Weeks {
		mikan.TotalItem += len(w.Items)
	}
	return mikan
}

func collectSeasons(root *html.Node, dateText string, out *[]model.MikanSeason) {
	forEachNode(root, func(n *html.Node) {
		if n.Type != html.ElementNode || n.Data != "li" {
			return
		}
		a := firstChild(n, "a")
		if a == nil {
			return
		}
		yearStr := attr(a, "data-year")
		seasonStr := attr(a, "data-season")
		if yearStr == "" || seasonStr == "" {
			return
		}
		year, _ := strconv.Atoi(yearStr)
		label := yearStr + " " + seasonStr
		*out = append(*out, model.MikanSeason{
			Year: year, Season: seasonStr, SeasonLabel: label,
			Select: strings.HasPrefix(dateText, label),
		})
	})
}

func collectAni(root *html.Node) []model.MikanInfo {
	var out []model.MikanInfo
	if root == nil {
		return out
	}
	for _, li := range childByTag(root, "li") {
		span := firstChild(li, "span")
		img := ""
		if span != nil {
			img = Host() + attr(span, "data-src")
		}
		as := childByTag(li, "a")
		if len(as) == 0 {
			continue
		}
		href := Host() + attr(as[0], "href")
		title := strings.TrimSpace(nodeText(as[0]))
		out = append(out, model.MikanInfo{
			Cover: img, Title: title, URL: href, Score: 0, Exists: false, BgmId: 0,
		})
	}
	return out
}

// GetMikanInfo parses a bangumi detail page.
func GetMikanInfo(bangumiId string) *model.MikanInfo {
	host := Host()
	url := host + "/Home/Bangumi/" + bangumiId
	info := &model.MikanInfo{URL: url}
	doc, err := fetch(url)
	if err != nil {
		return info
	}
	if cover := firstByClass(doc, "content"); cover != nil {
		if img := firstChild(cover, "img"); img != nil {
			info.Cover = host + attr(img, "src")
		}
	}
	if bt := firstByClass(doc, "bangumi-title"); bt != nil {
		info.Title = strings.TrimSpace(nodeText(bt))
	}
	for _, bi := range findByClass(doc, "bangumi-info") {
		if strings.TrimSpace(nodeOwnText(bi)) == "Bangumi番组计划链接：" {
			if a := firstChild(bi, "a"); a != nil {
				info.BgmUrl = attr(a, "href")
			}
		}
	}
	info.Groups = collectGroups(doc, host)
	return info
}

// GetGroups parses the subgroup list of a mikan page URL.
func GetGroups(rawURL string) []model.MikanGroup {
	doc, err := fetch(rawURL)
	if err != nil {
		return nil
	}
	host := Host()
	groups := collectGroups(doc, host)
	// group regex
	for i := range groups {
		groups[i].GroupRegex = groupregex.ToGroupRegex(titlesOf(groups[i].Items))
	}
	return groups
}

func collectGroups(doc *html.Node, host string) []model.MikanGroup {
	var groups []model.MikanGroup
	bgmUrl := ""
	for _, bi := range findByClass(doc, "bangumi-info") {
		if strings.TrimSpace(nodeOwnText(bi)) == "Bangumi番组计划链接：" {
			if a := firstChild(bi, "a"); a != nil {
				bgmUrl = attr(a, "href")
			}
		}
	}
	for _, item := range findByClass(doc, "leftbar-item") {
		g := model.MikanGroup{BgmUrl: bgmUrl}
		anchorName := ""
		if a := firstByClass(item, "subgroup-name"); a != nil {
			g.Label = strings.TrimSpace(nodeText(a))
			anchorName = attr(a, "data-anchor")
		}
		if anchorName != "" {
			anchorEl := findById(doc, strings.TrimPrefix(anchorName, "#"))
			if anchorEl != nil {
				if rss := firstByClass(anchorEl, "mikan-rss"); rss != nil {
					g.Rss = host + attr(rss, "href")
				}
				// items are in the following table
				table := nextSibling(anchorEl)
				if table != nil && table.Data == "table" {
					tbody := firstChild(table, "tbody")
					if tbody != nil {
						g.Items = collectItems(tbody, host)
					}
				}
			}
		}
		if day := firstByClass(item, "date"); day != nil {
			g.UpdateDay = strings.TrimSpace(nodeText(day))
		}
		groups = append(groups, g)
	}
	return groups
}

func collectItems(tbody *html.Node, host string) []model.MikanItem {
	var items []model.MikanItem
	for _, tr := range childByTag(tbody, "tr") {
		as := childByTag(tr, "a")
		if len(as) < 3 {
			continue
		}
		title := strings.TrimSpace(nodeOwnText(as[0]))
		magnet := attr(as[1], "data-clipboard-text")
		tds := childByTag(tr, "td")
		size := ""
		dateStr := ""
		if len(tds) >= 3 {
			size = strings.TrimSpace(nodeText(tds[2]))
			dateStr = strings.TrimSpace(nodeText(tds[3]))
		}
		torrent := host + attr(as[2], "href")
		items = append(items, model.MikanItem{
			Title: title, Magnet: magnet, FormatSize: size, Torrent: torrent,
			CreatedAt: model.DateTime(parseMikanDate(dateStr)),
		})
	}
	return items
}

func parseMikanDate(s string) time.Time {
	for _, layout := range []string{"2006-01-02 15:04", "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.ParseInLocation(layout, s, model.Loc()); err == nil {
			return t
		}
	}
	return time.Time{}
}

func titlesOf(items []model.MikanItem) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Title)
	}
	return out
}

// GetSubgroupId extracts the subgroupid query param from an RSS url.
func GetSubgroupId(rawURL string) string {
	if i := strings.Index(rawURL, "subgroupid="); i >= 0 {
		rest := rawURL[i+len("subgroupid="):]
		if j := strings.IndexAny(rest, "&?"); j >= 0 {
			rest = rest[:j]
		}
		return rest
	}
	return ""
}