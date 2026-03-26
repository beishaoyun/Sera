package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/sirupsen/logrus"
)

// TutorialScraper 教程抓取器
type TutorialScraper struct {
	httpClient *http.Client
	userAgent  string
}

// ScrapedContent 抓取内容
type ScrapedContent struct {
	URL         string            `json:"url"`
	Title       string            `json:"title"`
	Content     string            `json:"content"`
	HTML        string            `json:"html,omitempty"`
	Links       []string          `json:"links,omitempty"`
	Images      []string          `json:"images,omitempty"`
	CodeBlocks  []CodeBlock       `json:"code_blocks"`
	PublishDate string            `json:"publish_date,omitempty"`
	Author      string            `json:"author,omitempty"`
	Source      string            `json:"source"` // 来源类型
}

// CodeBlock 代码块
type CodeBlock struct {
	Language string `json:"language"`
	Code     string `json:"code"`
}

// NewTutorialScraper 创建教程抓取器
func NewTutorialScraper() *TutorialScraper {
	return &TutorialScraper{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: "Mozilla/5.0 (compatible; ServerMind/1.0; +https://github.com/servermind/aixm)",
	}
}

// Scrape 抓取网页内容
func (s *TutorialScraper) Scrape(ctx context.Context, pageURL string) (*ScrapedContent, error) {
	logrus.WithFields(logrus.Fields{
		"url": pageURL,
	}).Info("Scraping webpage")

	// 解析 URL 判断来源
	source := s.detectSource(pageURL)

	req, err := http.NewRequestWithContext(ctx, "GET", pageURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", s.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return s.parseHTML(pageURL, string(body), source)
}

// parseHTML 解析 HTML 内容
func (s *TutorialScraper) parseHTML(pageURL, html, source string) (*ScrapedContent, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	content := &ScrapedContent{
		URL:    pageURL,
		HTML:   html,
		Source: source,
	}

	// 提取标题
	content.Title = s.extractTitle(doc, source)

	// 提取作者
	content.Author = s.extractAuthor(doc, source)

	// 提取发布日期
	content.PublishDate = s.extractDate(doc, source)

	// 提取主要内容
	content.Content = s.extractContent(doc, source)

	// 提取代码块
	content.CodeBlocks = s.extractCodeBlocks(doc)

	// 提取链接
	content.Links = s.extractLinks(doc)

	// 提取图片
	content.Images = s.extractImages(doc, pageURL)

	return content, nil
}

// detectSource 检测来源
func (s *TutorialScraper) detectSource(pageURL string) string {
	u, err := url.Parse(pageURL)
	if err != nil {
		return "unknown"
	}

	host := strings.ToLower(u.Host)

	switch {
	case strings.Contains(host, "github.com"):
		return "github"
	case strings.Contains(host, "csdn.net"):
		return "csdn"
	case strings.Contains(host, "jianshu.com"):
		return "jianshu"
	case strings.Contains(host, "zhihu.com"):
		return "zhihu"
	case strings.Contains(host, "juejin.cn"):
		return "juejin"
	case strings.Contains(host, "baidu.com"):
		return "baidu"
	case strings.Contains(host, "google.com"):
		return "google"
	case strings.Contains(host, "medium.com"):
		return "medium"
	case strings.Contains(host, "dev.to"):
		return "dev.to"
	default:
		return "other"
	}
}

// extractTitle 提取标题
func (s *TutorialScraper) extractTitle(doc *goquery.Document, source string) string {
	switch source {
	case "github":
		// GitHub README 标题
		if title := doc.Find("h1").First(); title.Length() > 0 {
			return strings.TrimSpace(title.Text())
		}
	case "csdn":
		// CSDN 文章标题
		if title := doc.Find("h1.title-article"); title.Length() > 0 {
			return strings.TrimSpace(title.Text())
		}
	case "jianshu":
		// 简书文章标题
		if title := doc.Find("h1.title"); title.Length() > 0 {
			return strings.TrimSpace(title.Text())
		}
	case "juejin":
		// 掘金文章标题
		if title := doc.Find("h1.title"); title.Length() > 0 {
			return strings.TrimSpace(title.Text())
		}
	case "zhihu":
		// 知乎文章标题
		if title := doc.Find("h1.Post-Title"); title.Length() > 0 {
			return strings.TrimSpace(title.Text())
		}
	}

	// 默认提取 title 标签
	if title := doc.Find("title"); title.Length() > 0 {
		return strings.TrimSpace(title.Text())
	}

	return "Untitled"
}

// extractAuthor 提取作者
func (s *TutorialScraper) extractAuthor(doc *goquery.Document, source string) string {
	switch source {
	case "github":
		// GitHub 作者（仓库所有者）
		if author := doc.Find(".author a"); author.Length() > 0 {
			return strings.TrimSpace(author.Text())
		}
	case "csdn":
		// CSDN 作者
		if author := doc.Find("span.name"); author.Length() > 0 {
			return strings.TrimSpace(author.Text())
		}
	case "jianshu":
		// 简书作者
		if author := doc.Find("span.name"); author.Length() > 0 {
			return strings.TrimSpace(author.Text())
		}
	case "juejin":
		// 掘金作者
		if author := doc.Find("span.username"); author.Length() > 0 {
			return strings.TrimSpace(author.Text())
		}
	}
	return ""
}

// extractDate 提取日期
func (s *TutorialScraper) extractDate(doc *goquery.Document, source string) string {
	switch source {
	case "github":
		// GitHub 最后更新时间
		if date := doc.Find("relative-datetime"); date.Length() > 0 {
			return date.AttrOr("title", "")
		}
	case "csdn":
		// CSDN 发布时间
		if date := doc.Find("span.time"); date.Length() > 0 {
			return strings.TrimSpace(date.Text())
		}
	}
	return ""
}

// extractContent 提取主要内容
func (s *TutorialScraper) extractContent(doc *goquery.Document, source string) string {
	var contentBuilder strings.Builder

	switch source {
	case "github":
		// GitHub README 内容
		doc.Find(".Box-body article, .readme").Each(func(i int, sel *goquery.Selection) {
			contentBuilder.WriteString(sel.Text())
		})

	case "csdn":
		// CSDN 文章内容 (需要处理反爬虫)
		doc.Find("article.blog-content-pr, div.article_content").Each(func(i int, sel *goquery.Selection) {
			contentBuilder.WriteString(sel.Text())
		})

	case "jianshu":
		// 简书文章内容
		doc.Find("article").Each(func(i int, sel *goquery.Selection) {
			contentBuilder.WriteString(sel.Text())
		})

	case "juejin":
		// 掘金文章内容
		doc.Find("article.article-content").Each(func(i int, sel *goquery.Selection) {
			contentBuilder.WriteString(sel.Text())
		})

	default:
		// 通用提取：查找 main 标签或 article 标签
		contentSelectors := []string{
			"main article",
			"article",
			".content",
			".post-content",
			".article-body",
			"#content",
		}

		for _, selector := range contentSelectors {
			if sel := doc.Find(selector); sel.Length() > 0 {
				contentBuilder.WriteString(sel.Text())
				break
			}
		}

		// 如果还是没找到，提取所有段落
		if contentBuilder.Len() == 0 {
			doc.Find("p").Each(func(i int, sel *goquery.Selection) {
				if i < 50 { // 限制段落数量
					contentBuilder.WriteString(sel.Text())
					contentBuilder.WriteString("\n\n")
				}
			})
		}
	}

	return strings.TrimSpace(contentBuilder.String())
}

// extractCodeBlocks 提取代码块
func (s *TutorialScraper) extractCodeBlocks(doc *goquery.Document) []CodeBlock {
	var codeBlocks []CodeBlock

	// 提取 pre > code 结构的代码块
	doc.Find("pre").Each(func(i int, sel *goquery.Selection) {
		code := strings.TrimSpace(sel.Text())
		if code == "" {
			return
		}

		language := ""
		// 尝试从 class 中提取语言
		if class := sel.AttrOr("class", ""); class != "" {
			if matches := regexp.MustCompile(`language-(\w+)`).FindStringSubmatch(class); len(matches) > 1 {
				language = matches[1]
			}
		}

		// 检查子元素 code 标签
		sel.Find("code").Each(func(j int, codeSel *goquery.Selection) {
			if class := codeSel.AttrOr("class", ""); class != "" {
				if matches := regexp.MustCompile(`language-(\w+)`).FindStringSubmatch(class); len(matches) > 1 {
					language = matches[1]
				}
			}
		})

		codeBlocks = append(codeBlocks, CodeBlock{
			Language: language,
			Code:     code,
		})
	})

	return codeBlocks
}

// extractLinks 提取链接
func (s *TutorialScraper) extractLinks(doc *goquery.Document) []string {
	var links []string
	linkSet := make(map[string]bool)

	doc.Find("a[href]").Each(func(i int, sel *goquery.Selection) {
		if href, exists := sel.Attr("href"); exists {
			if !linkSet[href] && strings.HasPrefix(href, "http") {
				linkSet[href] = true
				links = append(links, href)
			}
		}
	})

	return links
}

// extractImages 提取图片
func (s *TutorialScraper) extractImages(doc *goquery.Document, baseURL string) []string {
	var images []string
	imageSet := make(map[string]bool)

	doc.Find("img[src]").Each(func(i int, sel *goquery.Selection) {
		src, _ := sel.Attr("src")
		if src == "" {
			return
		}

		// 处理相对路径
		if !strings.HasPrefix(src, "http") {
			base, _ := url.Parse(baseURL)
			if strings.HasPrefix(src, "/") {
				src = fmt.Sprintf("%s://%s%s", base.Scheme, base.Host, src)
			} else {
				src = fmt.Sprintf("%s://%s/%s", base.Scheme, base.Host, src)
			}
		}

		if !imageSet[src] {
			imageSet[src] = true
			images = append(images, src)
		}
	})

	return images
}

// ScrapeGitHubReadme 专门抓取 GitHub README
func (s *TutorialScraper) ScrapeGitHubReadme(ctx context.Context, repoURL string) (string, error) {
	// 解析 repo URL 获取 owner/repo
	owner, repo, branch, err := parseGitHubURL(repoURL)
	if err != nil {
		return "", err
	}

	// 构建 README 原始内容 URL
	readmeURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/README.md", owner, repo, branch)

	req, err := http.NewRequestWithContext(ctx, "GET", readmeURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", s.userAgent)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch README: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// 尝试 master 分支
		if branch != "master" {
			readmeURL = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/master/README.md", owner, repo)
			return s.ScrapeGitHubReadme(ctx, fmt.Sprintf("https://github.com/%s/%s", owner, repo))
		}
		return "", fmt.Errorf("README not found (status %d)", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// parseGitHubURL 解析 GitHub URL
func parseGitHubURL(repoURL string) (owner, repo, branch string, err error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", "", "", err
	}

	if !strings.Contains(u.Host, "github.com") {
		return "", "", "", fmt.Errorf("not a GitHub URL")
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return "", "", "", fmt.Errorf("invalid GitHub URL")
	}

	owner = parts[0]
	repo = parts[1]
	branch = "main"

	if len(parts) > 4 && parts[2] == "tree" {
		branch = parts[3]
	}

	return owner, repo, branch, nil
}
