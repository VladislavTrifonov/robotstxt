package robotstxt

import (
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"testing/quick"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatus(t *testing.T) {
	t.Parallel()
	type tcase struct {
		code  int
		allow bool
	}
	cases := []tcase{
		{200, true},
		{400, true},
		{401, true},
		{403, true},
		{404, true},
		{500, false},
		{502, false},
		{503, false},
		{504, false},
	}
	for _, c := range cases {
		t.Run(strconv.Itoa(c.code), func(t *testing.T) {
			r, err := FromResponse(newHttpResponse(c.code, ""))
			require.NoError(t, err)
			expectAll(t, r, c.allow)

			r, err = FromStatusAndString(c.code, "")
			require.NoError(t, err)
			expectAll(t, r, c.allow)
		})
	}
}

func TestFromStringDisallowAll(t *testing.T) {
	r, err := FromString("User-Agent: *\r\nDisallow: /\r\n")
	require.NoError(t, err)
	expectAll(t, r, false)
}

// missing user-agent implies all user agents.
func TestFromStringDisallowWPAdmin(t *testing.T) {
	r, err := FromString(`
Sitemap: http://www.tv-direct.net/sitemap_index.xml
Disallow:  /wp-admin`)
	require.NoError(t, err)
	expectAllAgents(t, r, false, "/wp-admin/")
}

// Test with HTML embeded in the text
func TestWithPairHtmlTags(t *testing.T) {
	r, err := FromString(`
<style>#wpadminbar {display:none !important;} html{margin-top:0px !important;} < /style>User-agent: *
Disallow: /wp-admin/
Allow: /wp-admin/admin-ajax.php

Sitemap: https://www.projectengineer.net/sitemap.xml`)
	require.NoError(t, err)
	expectAllAgents(t, r, false, "/wp-admin/")
	expectAllAgents(t, r, true, "/wp-admin/admin-ajax.php")
}

// Test with !DOCTYPE HTML embeded in the text
func TestWithSingleHtmlTags(t *testing.T) {
	r, err := FromString(`
<!DOCTYPE html>
User-agent: *
Disallow: /wp-admin/
Allow: /wp-admin/admin-ajax.php

Sitemap: https://www.projectengineer.net/sitemap.xml`)
	require.NoError(t, err)
	expectAllAgents(t, r, false, "/wp-admin/")
	expectAllAgents(t, r, true, "/wp-admin/admin-ajax.php")
}

func TestFromString002(t *testing.T) {
	t.Parallel()
	r, err := FromString("User-Agent: *\r\nDisallow: /account\r\n")
	require.NoError(t, err)
	expectAllAgents(t, r, true, "/foobar")
	expectAllAgents(t, r, false, "/account")
	expectAllAgents(t, r, false, "/account/sub")
}

const robotsText001 = "User-agent: * \nDisallow: /administrator/\nDisallow: /cache/\nDisallow: /components/\nDisallow: /editor/\nDisallow: /forum/\nDisallow: /help/\nDisallow: /images/\nDisallow: /includes/\nDisallow: /language/\nDisallow: /mambots/\nDisallow: /media/\nDisallow: /modules/\nDisallow: /templates/\nDisallow: /installation/\nDisallow: /getcid/\nDisallow: /tooltip/\nDisallow: /getuser/\nDisallow: /download/\nDisallow: /index.php?option=com_phorum*,quote=1\nDisallow: /index.php?option=com_phorum*phorum_query=search\nDisallow: /index.php?option=com_phorum*,newer\nDisallow: /index.php?option=com_phorum*,older\n\nUser-agent: Yandex\nAllow: /\nSitemap: http://www.pravorulya.com/sitemap.xml\nSitemap: http://www.pravorulya.com/sitemap1.xml"

func TestFromString003(t *testing.T) {
	t.Parallel()
	r, err := FromString(robotsText001)
	require.NoError(t, err)
	expectAllAgents(t, r, false, "/administrator/")
	expectAllAgents(t, r, true, "/paruram")
}

// Test with extra space before ":"
func TestFromString004(t *testing.T){
	t.Parallel()
	r, err := FromString(`
User-Agent: *
Crawl-delay : 60
Disallow : /*calendar*
Disallow : /*guestbook*`)
	require.NoError(t, err)
	expectAllAgents(t, r, false, "/*calendar*")
	expectAllAgents(t, r, false, "/*guestbook*")
}

// Test with Allow before a User-Agen
func TestFromString005(t *testing.T){
	t.Parallel()
	r, err := FromString(`
Allow: /wp-admin/admin-ajax.php
User-Agent: *
Allow: /wp-content/uploads/
Disallow: /wp-content/plugins/
Disallow: /wp-admin/
Disallow: /readme.html
Disallow: /refer/

Sitemap: https://www.mamatakecare/post-sitemap.xml
Sitemap: https://www.mamatakecare/page-sitemap.xml`)
	require.NoError(t, err)
	expectAllAgents(t, r, true, "/wp-content/uploads/")
	expectAllAgents(t, r, false, "/wp-content/plugins/")
	expectAllAgents(t, r, false, "/wp-admin/")
	expectAllAgents(t, r, false, "/readme.html")
	expectAllAgents(t, r, false, "/refer/")
}

// Test with a malformed value for Crawl-Delay
func TestFromString006(t *testing.T) {
	t.Parallel()
	r, err := FromString(`
# Make changes for all web spiders
User-agent: *
Crawl-delay: / `)
	require.NoError(t, err)
	expectCrawlDelay(t, r, "*", 0 * time.Second)
}

// Test with a valid Crawl-delay
func TestFromString007(t *testing.T) {
	t.Parallel()
	r, err := FromString(`
# Make changes for all web spiders
User-agent: bot
Crawl-delay: 100`)
	require.NoError(t, err)
	expectCrawlDelay(t, r, "bot", 100 * time.Second)
}

// Test with misspelling of user-agent
func TestFromString008(t *testing.T){
	t.Parallel()
	r, err := FromString(`
Usser-agent: *
Allow: /

Sitemap: https://www.prefix.ph/sitemap_index.xml`)
	require.NoError(t, err)
	expectAllAgents(t, r, true, "/")
}



// Test with misspelling of user-agent
func TestFromString009(t *testing.T){
	t.Parallel()
	r, err := FromString(`
#
ser-agent: Applebot
Allow: /`)
	require.NoError(t, err)
	expectAccess(t, r, true, "AppleBot", "/")
}

// Test with misspelling of user-agent
func TestFromString010(t *testing.T){
	t.Parallel()
	r, err := FromString(`
#
###################################################################################################################################

## GENERAL SETTINGS

User-agent: *


## SITEMAPS

# SITEMAP INDEX
Sitemap: https://qgear.es/sitemap_index.xml

# SITEMAP POST
Sitemap: https://qgear.es/post-sitemap.xml

# SITEMAP PAGINAS
Sitemap: https://qgear.es/page-sitemap.xml

# SITEMAP AMP
Sitemap:

# SITEMAP BLOG
Sitemap:
`)
	require.NoError(t, err)
	expectAllAgents(t, r, true, "/")
}

// Test with misspelling of user-agent
func TestFromString11(t *testing.T){
	t.Parallel()
	r, err := FromString(`
user-agent: Mozilla/5.0 (Linux; Android 8.0; Pixel 2 Build/OPD3.170816.012) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.121 Mobile Safari/537.36 (compatible; EzLynx/0.1; +http://www.mybot.com/bot.html)
Allow: /`)
	require.NoError(t, err)
	expectAccess(t, r, true, "/", "Mozilla/5.0 (Linux; Android 8.0; Pixel 2 Build/OPD3.170816.012) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.121 Mobile Safari/537.36 (compatible; EzLynx/0.1; +http://www.mybot.com/bot.html)")
}

// Test with Allow before a User-Agen
func TestFromString012(t *testing.T){
	t.Parallel()
	r, err := FromString(`
User-Agent: mybot
Allow: /photos/
Allow: /images/
Disallow: /
	
User-Agent: *
Allow: /photos/
Disallow: /`)
	require.NoError(t, err)
	expectNoAccess(t, r, true)
	expectAccess(t, r, true, "/photos/","mybot")
	expectAccess(t, r, true, "/images/","mybot")
	expectAccess(t, r, true, "/photos/","yourbot")
	expectAccess(t, r, false, "/","mybot")
	expectAccess(t, r, false, "/","yourbot")
}


func TestInvalidEncoding(t *testing.T) {
	// Invalid UTF-8 encoding should not break parser.
	_, err := FromString("User-agent: H\xef\xbf\xbdm�h�kki\nDisallow: *")
	require.NoError(t, err)
}

// http://www.google.com/robots.txt on Wed, 12 Jan 2011 12:22:20 GMT
const robotsGoogle = ("User-agent: *\nDisallow: /search\nDisallow: /groups\nDisallow: /images\nDisallow: /catalogs\nDisallow: /catalogues\nDisallow: /news\nAllow: /news/directory\nDisallow: /nwshp\nDisallow: /setnewsprefs?\nDisallow: /index.html?\nDisallow: /?\nDisallow: /addurl/image?\nDisallow: /pagead/\nDisallow: /relpage/\nDisallow: /relcontent\nDisallow: /imgres\nDisallow: /imglanding\nDisallow: /keyword/\nDisallow: /u/\nDisallow: /univ/\nDisallow: /cobrand\nDisallow: /custom\nDisallow: /advanced_group_search\nDisallow: /googlesite\nDisallow: /preferences\nDisallow: /setprefs\nDisallow: /swr\nDisallow: /url\nDisallow: /default\nDisallow: /m?\nDisallow: /m/?\nDisallow: /m/blogs?\nDisallow: /m/directions?\nDisallow: /m/ig\nDisallow: /m/images?\nDisallow: /m/local?\nDisallow: /m/movies?\nDisallow: /m/news?\nDisallow: /m/news/i?\nDisallow: /m/place?\nDisallow: /m/products?\nDisallow: /m/products/\nDisallow: /m/setnewsprefs?\nDisallow: /m/search?\nDisallow: /m/swmloptin?\nDisallow: /m/trends\nDisallow: /m/video?\nDisallow: /wml?\nDisallow: /wml/?\nDisallow: /wml/search?\nDisallow: /xhtml?\nDisallow: /xhtml/?\nDisallow: /xhtml/search?\nDisallow: /xml?\nDisallow: /imode?\nDisallow: /imode/?\nDisallow: /imode/search?\nDisallow: /jsky?\nDisallow: /jsky/?\nDisallow: /jsky/search?\nDisallow: /pda?\nDisallow: /pda/?\nDisallow: /pda/search?\nDisallow: /sprint_xhtml\nDisallow: /sprint_wml\nDisallow: /pqa\nDisallow: /palm\nDisallow: /gwt/\nDisallow: /purchases\nDisallow: /hws\nDisallow: /bsd?\nDisallow: /linux?\nDisallow: /mac?\nDisallow: /microsoft?\nDisallow: /unclesam?\nDisallow: /answers/search?q=\nDisallow: /local?\nDisallow: /local_url\nDisallow: /froogle?\nDisallow: /products?\nDisallow: /products/\nDisallow: /froogle_\nDisallow: /product_\nDisallow: /products_\nDisallow: /products;\nDisallow: /print\nDisallow: /books\nDisallow: /bkshp?q=\nAllow: /booksrightsholders\nDisallow: /patents?\nDisallow: /patents/\nAllow: /patents/about\nDisallow: /scholar\nDisallow: /complete\nDisallow: /sponsoredlinks\nDisallow: /videosearch?\nDisallow: /videopreview?\nDisallow: /videoprograminfo?\nDisallow: /maps?\nDisallow: /mapstt?\nDisallow: /mapslt?\nDisallow: /maps/stk/\nDisallow: /maps/br?\nDisallow: /mapabcpoi?\nDisallow: /maphp?\nDisallow: /places/\nAllow: /places/$\nDisallow: /maps/place\nDisallow: /help/maps/streetview/partners/welcome/\nDisallow: /lochp?\nDisallow: /center\nDisallow: /ie?\nDisallow: /sms/demo?\nDisallow: /katrina?\nDisallow: /blogsearch?\nDisallow: /blogsearch/\nDisallow: /blogsearch_feeds\nDisallow: /advanced_blog_search\nDisallow: /reader/\nAllow: /reader/play\nDisallow: /uds/\nDisallow: /chart?\nDisallow: /transit?\nDisallow: /mbd?\nDisallow: /extern_js/\nDisallow: /calendar/feeds/\nDisallow: /calendar/ical/\nDisallow: /cl2/feeds/\n" +
	"Disallow: /cl2/ical/\nDisallow: /coop/directory\nDisallow: /coop/manage\nDisallow: /trends?\nDisallow: /trends/music?\nDisallow: /trends/hottrends?\nDisallow: /trends/viz?\nDisallow: /notebook/search?\nDisallow: /musica\nDisallow: /musicad\nDisallow: /musicas\nDisallow: /musicl\nDisallow: /musics\nDisallow: /musicsearch\nDisallow: /musicsp\nDisallow: /musiclp\nDisallow: /browsersync\nDisallow: /call\nDisallow: /archivesearch?\nDisallow: /archivesearch/url\nDisallow: /archivesearch/advanced_search\nDisallow: /base/reportbadoffer\nDisallow: /urchin_test/\nDisallow: /movies?\nDisallow: /codesearch?\nDisallow: /codesearch/feeds/search?\nDisallow: /wapsearch?\nDisallow: /safebrowsing\nAllow: /safebrowsing/diagnostic\nAllow: /safebrowsing/report_error/\nAllow: /safebrowsing/report_phish/\nDisallow: /reviews/search?\nDisallow: /orkut/albums\nAllow: /jsapi\nDisallow: /views?\nDisallow: /c/\nDisallow: /cbk\nDisallow: /recharge/dashboard/car\nDisallow: /recharge/dashboard/static/\nDisallow: /translate_a/\nDisallow: /translate_c\nDisallow: /translate_f\nDisallow: /translate_static/\nDisallow: /translate_suggestion\nDisallow: /profiles/me\nAllow: /profiles\nDisallow: /s2/profiles/me\nAllow: /s2/profiles\nAllow: /s2/photos\nAllow: /s2/static\nDisallow: /s2\nDisallow: /transconsole/portal/\nDisallow: /gcc/\nDisallow: /aclk\nDisallow: /cse?\nDisallow: /cse/home\nDisallow: /cse/panel\nDisallow: /cse/manage\nDisallow: /tbproxy/\nDisallow: /imesync/\nDisallow: /shenghuo/search?\nDisallow: /support/forum/search?\nDisallow: /reviews/polls/\nDisallow: /hosted/images/\nDisallow: /ppob/?\nDisallow: /ppob?\nDisallow: /ig/add?\nDisallow: /adwordsresellers\nDisallow: /accounts/o8\nAllow: /accounts/o8/id\nDisallow: /topicsearch?q=\nDisallow: /xfx7/\nDisallow: /squared/api\nDisallow: /squared/search\nDisallow: /squared/table\nDisallow: /toolkit/\nAllow: /toolkit/*.html\nDisallow: /globalmarketfinder/\nAllow: /globalmarketfinder/*.html\nDisallow: /qnasearch?\nDisallow: /errors/\nDisallow: /app/updates\nDisallow: /sidewiki/entry/\nDisallow: /quality_form?\nDisallow: /labs/popgadget/search\nDisallow: /buzz/post\nDisallow: /compressiontest/\nDisallow: /analytics/reporting/\nDisallow: /analytics/admin/\nDisallow: /analytics/web/\nDisallow: /analytics/feeds/\nDisallow: /analytics/settings/\nDisallow: /alerts/\nDisallow: /phone/compare/?\nAllow: /alerts/manage\nSitemap: http://www.gstatic.com/s2/sitemaps/profiles-sitemap.xml\nSitemap: http://www.google.com/hostednews/sitemap_index.xml\nSitemap: http://www.google.com/ventures/sitemap_ventures.xml\nSitemap: http://www.google.com/sitemaps_webmasters.xml\nSitemap: http://www.gstatic.com/trends/websites/sitemaps/sitemapindex.xml\nSitemap: http://www.gstatic.com/dictionary/static/sitemaps/sitemap_index.xml")

func TestFromGoogle(t *testing.T) {
	t.Parallel()
	r, err := FromString(robotsGoogle)
	require.NoError(t, err)
	expectAllAgents(t, r, true, "/ncr")
	expectAllAgents(t, r, false, "/search")
}

func TestAllowAll(t *testing.T) {
	t.Parallel()
	cases := []string{
		"",
		"# comment",
		"User-Agent: * \nAllow: /",
		"User-Agent: * \nDisallow: ",
	}
	for i, input := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			r, err := FromString(input)
			require.NoError(t, err)
			expectAll(t, r, true)
		})
	}
}

func TestRobotstxtOrg(t *testing.T) {
	t.Parallel()
	const robotsText005 = `User-agent: Google
Disallow:
User-agent: *
Disallow: /`
	r, err := FromString(robotsText005)
	require.NoError(t, err)
	expectAccess(t, r, false, "/path/page1.html", "SomeBot")
	expectAccess(t, r, true, "/path/page1.html", "Googlebot")
}

func TestHost(t *testing.T) {
	type tcase struct {
		input  string
		expect string
	}
	cases := []tcase{
		{"#Host: site.ru", ""},
		{"Host: site.ru", "site.ru"},
		{"Host: яндекс.рф", "яндекс.рф"},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			r, err := FromString(c.input)
			require.NoError(t, err)
			assert.Equal(t, c.expect, r.Host)
		})
	}
}

func TestTheaid(t *testing.T) {
	const robotsTheaid = ("# Tier 0\nUser-agent: Googlebot\nUser-agent: Googlebot-Mobile\nUser-agent: Yandex # Russia\nDisallow: /admin/\nAllow: /\n#Crawl-delay: 1\n\n# Tier 1\nUser-agent: Bingbot # Bing\nUser-agent: Slurp # Yahoo\nUser-agent: coccoc # Vietnam\nUser-agent: DuckDuckBot # DuckDuckGo\nUser-agent: baiduspider # China\nUser-agent: Applebot # Apple\nDisallow: /admin/\nAllow: /\n#Crawl-delay: 10\n\n# Tier 1.5\nUser-agent: SeznamBot # Czech Republic\nUser-agent: NaverBot # South Korea\nUser-agent: Yeti # South Korea\nUser-agent: Mail.ru # Russia\nUser-agent: Sogou # China\nUser-agent: 360Spider # China\nDisallow: /admin/\nAllow: /\n#Crawl-delay: 30\n\n# Tier 2\nUser-agent: Teoma # Ask\n" +
		"User-agent: SputnikBot # Russia\nUser-agent: Daumoa # South Korea\nUser-agent: Dazoobot # France\nUser-agent: DeuSu # Germany\nUser-agent: EuripBot # Europe\nUser-agent: Exploratodo # Latin America\nUser-agent: istellabot # Italy\nUser-agent: moget # Japan\nUser-agent: ichiro # Japan\nUser-agent: Petalbot # Huawei\nUser-agent: Amazonbot\nUser-agent: Neevabot\nDisallow: /admin/\nAllow: /\n#Crawl-delay: 60\nCrawl-delay: 15\n\n# Tier 2.5\nUser-agent: Googlebot-News\nUser-agent: Googlebot-Video\nUser-agent: Googlebot-Image\nUser-agent: AdsBot-Google\nUser-agent: BingPreview\nUser-agent: msnbot\nUser-agent: msnbot-media\nDisallow: /admin/\nAllow: /\n#Crawl-delay: 90\n\n# Tier 3\n#User-agent: ccbot # commoncrawl.org\n#Disallow: /admin/\n#Allow: /\n#Crawl-delay: 120\n\n# Ads\nUser-agent: Mediapartners-Google\nUser-agent: ias_crawler\nUser-agent: GrapeshotCrawler\nUser-agent: CriteoBot\nUser-agent: TTD-Content\nUser-agent: weborama-fetcher\nUser-agent: AmazonAdBot\nDisallow: /admin/\nAllow: /\n\n# Blocked\nUser-agent: AhrefsBot\nUser-agent: SemrushBot\nUser-agent: BLEXBot\nUser-agent: MyBot\nDisallow: /\n\nUser-agent: *\nDisallow: /\n\nSitemap: https://www.chemicalaid.com/sitemap.php")

	t.Parallel()
	r, err := FromString(robotsTheaid)
	require.NoError(t, err)
	expectAccess(t, r, false, "/", "MyBot")
}

/*
// I don't want these to be errors, It's better to allow them with reasonable defaults.
// Than barf the rest of the file.
func TestParseErrors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		input  string
		expect string
	}{
		{"disallow-before", "Disallow: /\nUser-agent: bot", "Disallow before User-agent"},
		{"crawl-delay-syntax", "User-agent: bot\nCrawl-delay: bad-time-value", "invalid syntax"},
		{"crawl-delay-inf", "User-agent: bot\nCrawl-delay: -inf", "invalid value"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Log("input:", c.input)
			_, err := FromString(c.input)
			require.Error(t, err)
			_, ok := err.(*ParseError)
			assert.True(t, ok, "Expected ParseError")
			require.Contains(t, err.Error(), c.expect)
		})
	}
}
*/


const robotsTextJustHTML = `<!DOCTYPE html>
<html>
<title></title>
<p>Hello world! This is valid HTML but invalid robots.txt.`

func TestHtmlInstead(t *testing.T) {
	r, err := FromString(robotsTextJustHTML)
	// According to Google spec, invalid robots.txt file
	// must be parsed silently.
	require.NoError(t, err)
	group := r.FindGroup("SuperBot")
	require.NotNil(t, group)
	assert.True(t, group.Test("/"))
}

// http://perche.vanityfair.it/robots.txt on Sat, 13 Sep 2014 23:00:29 GMT
const robotsTextVanityfair = "\xef\xbb\xbfUser-agent: *\nDisallow: */oroscopo-di-oggi/*"

func TestWildcardPrefix(t *testing.T) {
	t.Parallel()
	r, err := FromString(robotsTextVanityfair)
	require.NoError(t, err)
	expectAllAgents(t, r, true, "/foo/bar")
	expectAllAgents(t, r, false, "/oroscopo-di-oggi/bar")
	expectAllAgents(t, r, false, "/foo/oroscopo-di-oggi/bar")
}

func TestGrouping(t *testing.T) {
	const robotsCaseGrouping = `user-agent: a
user-agent: b
disallow: /a
disallow: /b

user-agent: ignore
Disallow: /separator

user-agent: b
user-agent: c
disallow: /b
disallow: /c`

	r, err := FromString(robotsCaseGrouping)
	require.NoError(t, err)
	expectAccess(t, r, false, "/a", "a")
	expectAccess(t, r, false, "/b", "a")
	expectAccess(t, r, true, "/c", "a")

	expectAccess(t, r, false, "/a", "b")
	expectAccess(t, r, false, "/b", "b")
	expectAccess(t, r, false, "/c", "c")

	expectAccess(t, r, true, "/a", "c")
	expectAccess(t, r, false, "/b", "c")
	expectAccess(t, r, false, "/c", "c")
}

func BenchmarkParseFromString001(b *testing.B) {
	input := robotsText001
	b.ReportAllocs()
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := FromString(input); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseFromString002(b *testing.B) {
	input := robotsGoogle
	b.ReportAllocs()
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := FromString(input); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseFromStatus401(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := FromStatusAndString(401, ""); err != nil {
			b.Fatal(err)
		}
	}
}

func expectAll(t *testing.T, r *RobotsData, allow bool) {
	// TODO fuzz path
	expectAllAgents(t, r, allow, "/")
	expectAllAgents(t, r, allow, "/admin/")
	expectAllAgents(t, r, allow, "/search")
	expectAllAgents(t, r, allow, "/.htaccess")
}


func expectAllAgents(t *testing.T, r *RobotsData, allow bool, path string) {
	f := func(agent string) bool { return expectAccess(t, r, allow, path, agent) }
	if err := quick.Check(f, nil); err != nil {
		t.Fatalf("Expected allow path '%s' %v", path, err)
	}
}

func expectNoAccess(t *testing.T, r *RobotsData, allow bool) bool {
	return assert.Equal(t, allow, r.TestDisallowAll())
}

func expectAccess(t *testing.T, r *RobotsData, allow bool, path, agent string) bool {
	return assert.Equal(t, allow, r.TestAgent(path, agent), "path='%s' agent='%s'", path, agent)
}

func expectCrawlDelay(t *testing.T, r *RobotsData, agent string, delay time.Duration) bool {
	return assert.Equal(t, true, r.TestCrawlDelay(agent, delay), "agent='%s' delay='%v'", agent, delay)
}

func newHttpResponse(code int, body string) *http.Response {
	return &http.Response{
		StatusCode:    code,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Body:          ioutil.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
	}
}
