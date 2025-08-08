interface CardProps {
  href: string;
  title: string;
  body: string;
}

export default function Card({ href, title, body }: CardProps) {
  return (
    <li className="list-none flex p-0.5 bg-gradient-to-r from-purple-500 via-purple-300 to-[#23262d] bg-200% bg-pos-100 rounded-lg shadow-lg transition-all duration-500 ease-out hover:bg-pos-0 hover:shadow-2xl">
      <a href={href} className="w-full no-underline leading-normal p-6 rounded-md text-white bg-[#23262d] opacity-95 transition-colors duration-300 ease-in-out">
        <h2 className="transition-colors duration-300 ease-in-out text-white hover:text-purple-300">
          {title}
          <span className="inline-block transition-transform duration-300 ease-in-out transform group-hover:translate-x-1">&rarr;</span>
        </h2>
        <p>{body}</p>
      </a>
    </li>
  );
}
