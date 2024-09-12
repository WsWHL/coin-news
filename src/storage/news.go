package storage

import (
	"github.com/gin-gonic/gin"
	"news/src/models"
	"news/src/utils"
)

// NewsService 资讯服务
type NewsService struct {
	store *Service
}

// NewNewsService creates a new NewsService
func NewNewsService() *NewsService {
	return &NewsService{
		store: NewService(),
	}
}

func (s *NewsService) Release() {
	s.store.Release()
}

// HomeHandler 主页
func (s *NewsService) HomeHandler(c *utils.ApiContext) {
	req := struct {
		Category string `form:"category"`
		Page     int    `form:"page,default=1" binding:"required"`
		PageSize int    `form:"page_size,default=15" binding:"required"`
		Lang     string `form:"lang,default=en"`
	}{}
	if err := c.ShouldBind(&req); err != nil {
		c.Error(400, "参数错误")
		return
	}

	list, total := s.store.GetHomeList(req.Category, req.Page, req.PageSize)

	articles := make([]articleInfo, 0, len(list))
	for _, article := range list {
		articles = append(articles, newArticleInfo(article, req.Lang))
	}

	c.Pager(int(total), req.Page, req.PageSize, articles)
}

// HomeListHandler 主页完整列表
func (s *NewsService) HomeListHandler(c *utils.ApiContext) {
	category := c.Param("category")
	lang := c.Param("lang")

	list, err := s.store.GetListByCategory(category)
	if err != nil {
		c.Error(404, "资源未找到")
		return
	}

	articles := make([]articleInfo, 0, len(list))
	for _, article := range list {
		articles = append(articles, newArticleInfo(article, lang))
	}

	c.Ok(articles)
}

// HomeLinkHandler 文章链接
func (s *NewsService) HomeLinkHandler(c *utils.ApiContext) {
	token := c.Param("token")
	article, err := s.store.Get(token)
	if err != nil {
		c.Error(404, "资源未找到")
		return
	}

	c.Ok(gin.H{
		"link": article.Link,
	})
}

// HomeOriginListHandler 分类来源列表
func (s *NewsService) HomeOriginListHandler(c *utils.ApiContext) {
	category := c.PostForm("category")
	origins, err := s.store.GetOriginsByCategory(category)
	if err != nil {
		c.Error(500, "分类不存在")
		return
	}

	c.Ok(origins)
}

// NewsReadListHandler 资讯读取列表
func (s *NewsService) NewsReadListHandler(c *utils.ApiContext) {
	req := struct {
		Origins  []string `form:"origins"`
		Category string   `form:"category"`
		Lang     string   `form:"lang,default=en"`
	}{}
	if err := c.ShouldBind(&req); err != nil {
		c.Error(400, "参数错误")
		return
	}

	list, err := s.store.GetReadList(req.Origins, req.Category)
	if err != nil {
		c.Error(500, "获取文章列表失败")
		return
	}

	articles := make(map[string][]articleInfo)
	for key, items := range list {
		articles[key] = make([]articleInfo, 0, len(items))
		for _, article := range items {
			articles[key] = append(articles[key], newArticleInfo(article, req.Lang))
		}
	}

	c.Ok(articles)
}

// NewsOriginListHandler 资讯来源网站文章列表
func (s *NewsService) NewsOriginListHandler(c *utils.ApiContext) {
	origin := c.Param("origin")
	params := struct {
		Page     int    `form:"page,default=1" binding:"gt=0"`
		PageSize int    `form:"page_size,default=15" binding:"gt=0"`
		Lang     string `form:"lang"`
	}{}
	if err := c.ShouldBind(&params); err != nil {
		c.Error(400, "参数错误")
		return
	}

	list, count, err := s.store.GetListByOrigin(origin, params.Page, params.PageSize)
	if err != nil {
		c.Error(500, "获取文章列表失败")
		return
	}

	articles := make([]articleInfo, 0, len(list))
	for _, article := range list {
		articles = append(articles, newArticleInfo(article, params.Lang))
	}

	c.Pager(int(count), params.Page, params.PageSize, articles)
}

// NewsSearchHandler 资讯搜索
func (s *NewsService) NewsSearchHandler(c *utils.ApiContext) {
	req := struct {
		Keyword  string `form:"keyword"`
		Lang     string `form:"lang"`
		Page     int    `form:"page,default=1" binding:"gt=0"`
		PageSize int    `form:"page_size,default=15" binding:"gt=0"`
	}{}
	if err := c.ShouldBind(&req); err != nil {
		c.Error(400, "参数错误")
		return
	}

	list, count := s.store.NewsSearch(req.Keyword, req.Page, req.PageSize)
	articles := make([]articleInfo, 0, len(list))
	for _, article := range list {
		articles = append(articles, newArticleInfo(article, req.Lang))
	}

	c.Pager(int(count), req.Page, req.PageSize, articles)
}

// 文章信息
type articleInfo struct {
	From     string `json:"from"`
	Datetime string `json:"datetime"`
	Title    string `json:"title"`
	Link     string `json:"link"`
	Author   string `json:"author"`
	Image    string `json:"image"`
	Token    string `json:"token"`
	Abstract string `json:"abstract"`
}

func newArticleInfo(article *models.Article, lang string) articleInfo {
	return articleInfo{
		From:     article.From,
		Datetime: article.PubDate.Time.Format("2006-01-02 15:04:05"),
		Title:    article.GetTitleByLang(lang),
		Link:     article.Link,
		Author:   article.Author,
		Image:    article.Image,
		Token:    article.Token,
		Abstract: article.Abstract,
	}
}
