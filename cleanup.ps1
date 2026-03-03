docker system prune -a --volumes -f
python -m pip cache purge
conda clean --all -y
go clean -cache
go clean -modcache
Remove-Item -Path $env:TEMP\* -Recurse -Force -ErrorAction SilentlyContinue
