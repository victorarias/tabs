import * as React from 'react'

type Props = {
  code: string
  language?: string
}

let highlighterPromise: Promise<any> | null = null

async function getHighlighter() {
  if (!highlighterPromise) {
    highlighterPromise = import('shiki').then(({ createHighlighter }) =>
      createHighlighter({
        themes: ['github-light', 'github-dark'],
        langs: ['json', 'bash', 'text'],
      }),
    )
  }
  return highlighterPromise
}

export function CodeBlock({ code, language = 'json' }: Props) {
  const [html, setHtml] = React.useState<string | null>(null)

  React.useEffect(() => {
    let active = true
    setHtml(null)
    const run = async () => {
      try {
        const highlighter = await getHighlighter()
        const rendered = highlighter.codeToHtml(code, {
          lang: language || 'text',
          themes: { light: 'github-light', dark: 'github-dark' },
        })
        if (active) {
          setHtml(rendered)
        }
      } catch {
        if (active) {
          setHtml(null)
        }
      }
    }
    run()
    return () => {
      active = false
    }
  }, [code, language])

  if (!code) {
    return null
  }

  if (html) {
    return <div className="code-block" dangerouslySetInnerHTML={{ __html: html }} />
  }

  return (
    <pre className="code-block">
      <code>{code}</code>
    </pre>
  )
}
