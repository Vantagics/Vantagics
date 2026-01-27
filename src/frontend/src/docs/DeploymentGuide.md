# Deployment Guide - Dashboard Drag-Drop Layout

## Overview

This guide covers the deployment process for the Dashboard Drag-Drop Layout feature, including build procedures, environment setup, and rollout strategies.

## Prerequisites

### Development Environment
- Node.js 18+ 
- Go 1.21+
- Git
- Modern web browser for testing

### Production Environment
- Web server (nginx/Apache) or CDN
- Database (SQLite/PostgreSQL)
- SSL certificate for HTTPS
- Monitoring tools

## Build Process

### 1. Frontend Build
```bash
cd src/frontend
npm install
npm run build
```

**Output**: `dist/` directory with optimized assets
- `index.html` - Main application entry point
- `assets/` - Minified JS/CSS bundles with cache-busting hashes

### 2. Backend Build
```bash
cd src
go mod tidy
go build -o rapidbi.exe
```

**Output**: Executable binary with embedded frontend assets

### 3. Full Application Build
```bash
# Windows
build.bat

# Unix/Linux/macOS
./build.sh
```

## Deployment Strategies

### Strategy 1: Direct Deployment (Recommended for MVP)
1. Build production bundle
2. Deploy entire application as single binary
3. Update database schema if needed
4. Restart application service

### Strategy 2: Gradual Rollout (Future Enhancement)
1. Deploy with feature flag disabled
2. Enable for beta users first
3. Monitor metrics and feedback
4. Gradually increase rollout percentage
5. Full deployment when stable

### Strategy 3: Blue-Green Deployment (Enterprise)
1. Deploy to green environment
2. Run smoke tests
3. Switch traffic from blue to green
4. Keep blue as rollback option

## Environment Configuration

### Development
```json
{
  "environment": "development",
  "debug": true,
  "database": "dev.db",
  "features": {
    "dragDropLayout": true,
    "propertyBasedTesting": true
  }
}
```

### Staging
```json
{
  "environment": "staging",
  "debug": false,
  "database": "staging.db",
  "features": {
    "dragDropLayout": true,
    "propertyBasedTesting": false
  }
}
```

### Production
```json
{
  "environment": "production",
  "debug": false,
  "database": "production.db",
  "features": {
    "dragDropLayout": true,
    "propertyBasedTesting": false
  }
}
```

## Database Migration

### Migration Script
```sql
-- Add layout_configs table if not exists
CREATE TABLE IF NOT EXISTS layout_configs (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    config TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Add indexes for performance
CREATE INDEX IF NOT EXISTS idx_layout_configs_name ON layout_configs(name);
CREATE INDEX IF NOT EXISTS idx_layout_configs_created_at ON layout_configs(created_at);
```

### Migration Validation
```bash
# Run migration verification
go run src/database/verify_migration.go
```

## Deployment Checklist

### Pre-Deployment
- [ ] All tests passing (unit, integration, property-based)
- [ ] Code review completed
- [ ] Documentation updated
- [ ] Database migration script prepared
- [ ] Rollback plan documented
- [ ] Monitoring alerts configured

### Deployment Steps
1. [ ] Create deployment branch from main
2. [ ] Run full test suite
3. [ ] Build production bundle
4. [ ] Backup current database
5. [ ] Deploy to staging environment
6. [ ] Run smoke tests on staging
7. [ ] Deploy to production environment
8. [ ] Run database migrations
9. [ ] Verify application startup
10. [ ] Run post-deployment tests
11. [ ] Monitor application metrics
12. [ ] Update deployment documentation

### Post-Deployment
- [ ] Monitor error rates and performance
- [ ] Verify all features working correctly
- [ ] Check user feedback and support tickets
- [ ] Document any issues and resolutions
- [ ] Plan next iteration improvements

## Rollback Procedures

### Immediate Rollback (< 5 minutes)
1. Revert to previous application binary
2. Restart application service
3. Verify functionality restored

### Database Rollback (if needed)
1. Stop application
2. Restore database from backup
3. Deploy previous application version
4. Restart and verify

### Feature Flag Rollback
1. Disable `dragDropLayout` feature flag
2. Application falls back to previous layout system
3. No restart required

## Monitoring and Alerts

### Key Metrics to Monitor
- Application startup time
- Dashboard load performance
- Drag-drop operation latency
- Database query performance
- Error rates and exceptions
- User engagement with new features

### Alert Thresholds
- Error rate > 1%
- Response time > 2 seconds
- Database connection failures
- Memory usage > 80%
- CPU usage > 70%

## Troubleshooting

### Common Issues

#### 1. Bundle Loading Errors
**Symptoms**: White screen, console errors about missing assets
**Solution**: 
- Check asset paths in production build
- Verify web server configuration
- Clear browser cache

#### 2. Database Migration Failures
**Symptoms**: Application fails to start, database errors
**Solution**:
- Check database permissions
- Verify migration script syntax
- Restore from backup if needed

#### 3. Drag-Drop Not Working
**Symptoms**: Components not draggable, layout not saving
**Solution**:
- Check browser compatibility
- Verify JavaScript bundle loaded correctly
- Check for console errors

#### 4. Performance Issues
**Symptoms**: Slow loading, laggy interactions
**Solution**:
- Check bundle size and loading times
- Monitor database query performance
- Verify CDN configuration

## Security Considerations

### Frontend Security
- Content Security Policy (CSP) headers
- HTTPS enforcement
- XSS protection
- Asset integrity checks

### Backend Security
- Input validation and sanitization
- SQL injection prevention
- Authentication and authorization
- Rate limiting

## Performance Optimization

### Frontend Optimizations
- Bundle splitting and lazy loading
- Image optimization and compression
- CDN for static assets
- Browser caching strategies

### Backend Optimizations
- Database indexing
- Query optimization
- Connection pooling
- Caching strategies

## Support and Maintenance

### Documentation
- User guide for new features
- Developer documentation for extensions
- API documentation for integrations
- Troubleshooting guides

### Maintenance Schedule
- Weekly performance reviews
- Monthly security updates
- Quarterly feature assessments
- Annual architecture reviews

---

**Deployment Status**: READY  
**Last Updated**: January 19, 2026  
**Version**: 1.0.0