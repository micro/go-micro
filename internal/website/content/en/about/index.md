---
title: About Goldydocs
linkTitle: About
description: A sample site using the Docsy Hugo theme.
# menu: { main: { weight: 10 } }
---
{{% blocks/cover
  title="About Goldydocs"
  height="auto td-below-navbar"
  image_anchor="bottom"
%}}
{{% _param description %}}
{.display-6}
{{% /blocks/cover %}}

{{< blocks/section color="dark" type="row" >}}
{{% blocks/feature icon="fa-lightbulb" title="Fastest OS **on the planet**!" %}}
The new **TechOS** operating system is an open source project. It is a new project, but with grand ambitions.
Please follow this space for updates!
{{% /blocks/feature %}}
{{% blocks/feature icon="fa-brands fa-github" title="Contributions welcome!" url="https://github.com/gohugoio/hugo" %}}
We do a [Pull Request](https://github.com/gohugoio/hugo/pulls) contributions workflow on **GitHub**. New users are always welcome!
{{% /blocks/feature %}}
{{% blocks/feature icon="fa-brands fa-x-twitter" title="Follow us on Twitter!" url="https://twitter.com/GoHugoIO" %}}
For announcement of latest features etc.
{{% /blocks/feature %}}
{{< /blocks/section >}}

{{% blocks/lead color="white" %}}
Goldydocs is a sample site using the [Docsy](https://github.com/google/docsy)
Hugo theme that shows what it can do and provides you with a template site
structure. It’s designed for you to clone and edit as much as you like. See the
different sections of the documentation and site for more ideas.
{{% /blocks/lead %}}

{{% blocks/section type="text-center h1 py-4" %}}
This is another section with center alignment
{{% /blocks/section %}}

{{% blocks/section type="container h1 py-4" %}}
This is another section with default alignment
{{% /blocks/section %}}

{{% alert title="Welcome" %}} **Hello**, world! {{% /alert %}}


- The following note is part of this list item:
  {{% alert title="Celebrate!" color=success %}}
  This alert is properly indented and rendered as part of the list item. Notice
  how a Markdown [link definition][] gets resolved even if it is defined
  _outside_ of the alert body.

  > Nested shortcode used here → Hello, world!
  {{% /alert %}}
  The first list item content continues here.

- **Don't put content on the same line** as the opening tag, it breaks rendering:
  {{% alert title="This alert's rendering is broken!" color=warning %}} **Notice
  how the alert content appears outside of the list!** {{% /alert %}}

[link definition]: # 'A link definition defined outside the alert body.'

{{% conditional-text include-if="foo" %}}
This text appears in the output only if `buildCondition = "foo" is set in your config file`.
{{% /conditional-text %}}
{{% conditional-text exclude-if="bar" %}}
This text does not appear in the output if `buildCondition = "bar" is set in your config file`.
{{% /conditional-text %}}

{{< readfile file="/data/vanity.yaml" code="true" lang="yaml" draft="true" >}}

