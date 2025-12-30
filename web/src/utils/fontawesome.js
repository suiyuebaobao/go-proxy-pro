let loadingPromise = null

export function ensureFontAwesomeLoaded() {
  if (loadingPromise) return loadingPromise

  loadingPromise = Promise.all([
    import('@fortawesome/fontawesome-free/css/fontawesome.min.css'),
    import('@fortawesome/fontawesome-free/css/solid.min.css'),
    import('@fortawesome/fontawesome-free/css/brands.min.css')
  ])

  // 永远不抛到全局，避免影响首屏
  loadingPromise.catch(() => {})
  return loadingPromise
}

