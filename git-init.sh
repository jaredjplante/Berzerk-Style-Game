git init
git config user.email "jaredjplante@gmail.com"
git config user.name "jaredjplante"
git add .
echo 'Enter a commit comment'
read commitComment
git commit -m "$commitComment"
echo 'Enter the remote origin URl'
read url
git remote add origin "$url"
git branch -m main
git push origin main
