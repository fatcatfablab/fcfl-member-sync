# fcfl-member-sync

FatCatFabLab's member syncing facilities.

## How to create a release

Github Actions will automatically create release tarballs when a git tag is
pushed. Thus creating a release goes something like this:
1. Make changes
1. Commit & push: `git commit -m "Made some changes" && git push`
1. Create a tag and push it: `git tag v6.6.6 -m "My changes" && git push --tag`

For instructions to deploy the release, look at
https://github.com/fatcatfablab/fcfl-ansible
