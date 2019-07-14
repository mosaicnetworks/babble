# Building the docs

```bash
(mkvirtualenv -r requirements.txt docs)
workon docs
make html
google-chrome _build/html/index.html
deactivate
```
