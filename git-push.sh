git add .
echo 'Enter a commit comment: '
read commitComment
git commit -m "$commitComment"
echo 'What branch should the commit be pushed to?: '
read branch
git push origin "$branch"