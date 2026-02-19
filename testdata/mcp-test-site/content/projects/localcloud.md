---
title: "LocalCloud"
date: 2024-09-01T10:00:00Z
draft: false
tags:
  - go
  - aws
  - infrastructure
categories:
  - Infrastructure
description: "A local cloud service mock for development and testing."
summary: "Mock cloud services locally."
layout: project
---

## Overview

LocalCloud mocks cloud API calls locally for development and testing. This isn't a novel idea — [LocalStack](https://localstack.cloud/) already does this and does it well. But sometimes it's just fun to build something yourself, learn how the APIs actually work under the hood, and shape the tool to fit your own workflows.

## Features

- S3 compatible API
- Compatible with AWS SDK

## Current Status

Compute services are not supported yet. Storage and API mocking are the current focus. On the roadmap, EKS and GKE support is an interesting possibility — the idea would be to back those APIs with a local [k3s](https://k3s.io/) cluster, giving you a lightweight but real Kubernetes environment. Lambda/Cloud Functions and container services like ECS/GCS are lower priority for now.

## Tech Stack

Go, Docker.
