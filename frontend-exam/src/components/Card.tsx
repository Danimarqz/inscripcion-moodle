interface CardProps {
  href: string;
  title: string;
  body: string;
}

export default function Card({ href, title, body }: CardProps) {
  return (
    <li className="link-card">
      <a href={href}>
        <h2>
          {title}
          <span>&rarr;</span>
        </h2>
        <p>{body}</p>
      </a>
    </li>
  );
}
