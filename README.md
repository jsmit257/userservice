## userservice

This sorta provides an AuthN service, i.e.: login plus some profile info; the idea is 
that it would interface with an RBAC service, maybe billing, etc to make a generic user
platform. It's really just an experiment of standing up a data-agnostic service from 
soup to nuts, with proper testing, logging, metrics, etc