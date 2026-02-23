+++
title = "Styling Standards"
author = ["readb5@cardiff.ac.uk"]
date = "2026-02-23"
+++

## Summary

## Problem

In the previous website development project (with Benjy, Ed and Antonio), a huge amount of merge conflicts were caused
by a large single, global, stylesheet, since every front-end merge request would change this file in different ways,
adding new classes and such.

This generally is caused since having a global stylesheet is designed to have lots of shared styles across pages. In
practice however, every developer makes most of their own styles for their own pages, and so the global stylesheet
becomes a dumping ground for all styles, which causes merge conflicts.

## Options Considered

### Use a global stylesheet, and encourage more shared styles between web pages

I don't personally see a reason to limit the options a developer has to create their own styles, and I don't see a
reason to encourage more shared styles between web pages, since every page is different and has different styling needs.
Especially since this is a mostly backend-focused project, I don't think it's worth the effort to try to encourage more
shared styles between web pages.

## Recommendation

### Use a separate stylesheet for each page, and only use a global stylesheet for truly shared styles (primarily colour variables)

This is the option I recommend, since it allows developers to have more freedom to create their own styles for their own
pages, without worrying about merge conflicts in a global stylesheet. There is not any significant overhead or downside
to having separate stylesheets for each page, and actually might make it easier for developers to find and edit the
styles for their own pages, since they won't have to sift through a large global stylesheet to find the styles for their
page.

Me and Ed have discussed, based on the client feedback from
[#29](https://git.cardiff.ac.uk/c23041974/secure-package-registry/-/issues/29), that we need to standardise our styling
somewhat, since our about us and landing page looked inconsistent. Therefore we will be using a global stylesheet for
truly shared styles, primarily colour variables, and then each page will have its own stylesheet for the styles of that
page.
