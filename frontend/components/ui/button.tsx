import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

export function Button({ children, ...props }: any) {
  return (
    <button
      className="inline-flex items-center justify-center rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 bg-primary text-primary-foreground hover:bg-primary/90 h-10 px-4 py-2"
      {...props}
    >
      {children}
    </button>
  )
}

export function Card({ className, ...props }: any) {
  return (
    <div className="rounded-lg border bg-card text-card-foreground shadow-sm" {...props} />
  )
}

export function CardHeader({ className, ...props }: any) {
  return <div className="flex flex-col space-y-1.5 p-6" {...props} />
}

export function CardTitle({ className, ...props }: any) {
  return <h3 className="text-2xl font-semibold leading-none tracking-tight" {...props} />
}

export function CardDescription({ className, ...props }: any) {
  return <p className="text-sm text-muted-foreground mt-2" {...props} />
}

export function CardContent({ className, ...props }: any) {
  return <div className="p-6 pt-0" {...props} />
}
