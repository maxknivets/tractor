#!/usr/bin/env bash
nc -w 0 -U ~/.tractor/agent.sock || rm ~/.tractor/agent.sock