package plugins

// Taken verbatim from https://github.com/jekyll/jekyll-seo-tag/
// Adapted from github.com/jekyll/jekyll-seo-tag. Used according to the MIT License.
const seoTagTemplateSource = `{% if seo_tag.title? %}
  <title>{{ seo_tag.title }}</title>
{% endif %}

<meta name="generator" content="Jekyll v{{ jekyll.version }}" />

{% if seo_tag.page_title %}
  <meta property="og:title" content="{{ seo_tag.page_title }}" />
{% endif %}

{% if seo_tag.author.name %}
  <meta name="author" content="{{ seo_tag.author.name }}" />
{% endif %}

<meta property="og:locale" content="{{ seo_tag.page_locale | replace:'-','_' }}" />

{% if seo_tag.description %}
  <meta name="description" content="{{ seo_tag.description | escape }}" />
  <meta name="twitter:description" property="og:description" content="{{ seo_tag.description | escape }}" />
{% endif %}
{% if seo_tag.canonical_url %}
  <link rel="canonical" href="{{ seo_tag.canonical_url }}" />
  <meta property="og:url" content="{{ seo_tag.canonical_url }}" />
{% endif %}

{% if seo_tag.site_title %}
  <meta property="og:site_name" content="{{ seo_tag.site_title }}" />
{% endif %}

{% if seo_tag.image %}
  <meta property="og:image" content="{{ seo_tag.image.path }}" />
  {% if seo_tag.image.height %}
    <meta property="og:image:height" content="{{ seo_tag.image.height }}" />
  {% endif %}
  {% if seo_tag.image.width %}
    <meta property="og:image:width" content="{{ seo_tag.image.width }}" />
  {% endif %}
{% endif %}

{% if seo_tag.date_published %}
  <meta property="og:type" content="article" />
  <meta property="article:published_time" content="{{ seo_tag.date_published | date_to_xmlschema }}" />
  {% if seo_tag.date_modified %}
    <meta property="article:modified_time" content="{{ seo_tag.date_modified | date_to_xmlschema }}" />
  {% endif %}
{% else %}
  <meta property="og:type" content="website" />
{% endif %}

{% if paginator.previous_page %}
  <link rel="prev" href="{{ paginator.previous_page_path | absolute_url }}">
{% endif %}
{% if paginator.next_page %}
  <link rel="next" href="{{ paginator.next_page_path | absolute_url }}">
{% endif %}

{% if seo_tag.image %}
  <meta name="twitter:card" content="summary_large_image" />
{% else %}
  <meta name="twitter:card" content="summary" />
{% endif %}
{% if seo_tag.page_title %}
  <meta name="twitter:title" content="{{ seo_tag.page_title }}" />
{% endif %}

{% if site.twitter.username %}
  <meta name="twitter:site" content="@{{ site.twitter.username | replace:"@","" }}" />
{% endif %}

{% if seo_tag.author.twitter %}
  <meta name="twitter:creator" content="@{{ seo_tag.author.twitter }}" />
{% endif %}

{% if site.facebook %}
  {% if site.facebook.admins %}
    <meta property="fb:admins" content="{{ site.facebook.admins }}" />
  {% endif %}

  {% if site.facebook.publisher %}
    <meta property="article:publisher" content="{{ site.facebook.publisher }}" />
  {% endif %}

  {% if site.facebook.app_id %}
    <meta property="fb:app_id" content="{{ site.facebook.app_id }}" />
  {% endif %}
{% endif %}

{% if site.webmaster_verifications %}
  {% if site.webmaster_verifications.google %}
    <meta name="google-site-verification" content="{{ site.webmaster_verifications.google }}">
  {% endif %}

  {% if site.webmaster_verifications.bing %}
    <meta name="msvalidate.01" content="{{ site.webmaster_verifications.bing }}">
  {% endif %}

  {% if site.webmaster_verifications.alexa %}
    <meta name="alexaVerifyID" content="{{ site.webmaster_verifications.alexa }}">
  {% endif %}

  {% if site.webmaster_verifications.yandex %}
    <meta name="yandex-verification" content="{{ site.webmaster_verifications.yandex }}">
  {% endif %}
{% elsif site.google_site_verification %}
  <meta name="google-site-verification" content="{{ site.google_site_verification }}" />
{% endif %}

<script type="application/ld+json">
  {{ seo_tag.json_ld | jsonify }}
</script>`
