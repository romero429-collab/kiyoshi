# Railway Deployment Guide

## Quick Deploy to Railway

### Option 1: Connect GitHub (Easiest)

1. Go to https://railway.app
2. Sign in with GitHub
3. Click "New Project" → "Deploy from GitHub repo"
4. Select `romero429-collab/kiyoshi`
5. Railway auto-detects the Dockerfile
6. Click "Deploy"
7. Your app is live! 🚀

### Option 2: Railway CLI

```bash
# Install Railway CLI
npm i -g @railway/cli

# Login
railway login

# Init project
cd kiyoshi
railway init

# Deploy
railway up
```

---

## Environment Variables

Set in Railway dashboard:

```
CLI_MODE=http
CLI_ADDR=:8080
REACT_APP_API_URL=https://<your-railway-domain>.railway.app
```

---

## Testing After Deploy

1. Get your Railway URL from the dashboard
2. Open on your phone: `https://<your-railway-domain>.railway.app`
3. Type a task
4. Click Send
5. Watch it execute! ✨

---

## Monitoring

In Railway dashboard:
- View logs: Real-time server logs
- View metrics: CPU, memory, network
- View deployments: Deployment history

---

## Cost

- **Free tier**: $5 credit/month (usually enough for light usage)
- **Pay-as-you-go**: After credits, ~$0.25/hour for a basic instance

---

## Troubleshooting

**App won't deploy:**
- Check logs in Railway dashboard
- Ensure `Dockerfile` exists in root
- Verify environment variables are set

**API connection fails:**
- Check that both services (CLI + Web) are running
- Verify `REACT_APP_API_URL` is correct
- Check browser console for CORS errors

**Slow response:**
- Railway free tier is single CPU
- Consider upgrading to paid tier for better performance

---

Enjoy Kiyoshi on your phone! 📱✨
