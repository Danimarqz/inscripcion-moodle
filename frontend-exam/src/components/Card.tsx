interface CardProps {
  href: string;
  title: string;
  body: string;
}

export default function Card({ href, title, body }: CardProps) {
  return (
    <li className="group list-none flex p-0.5 bg-gradient-to-r from-brand-pink via-brand-yellow to-brand-blue bg-200% bg-pos-100 rounded-lg shadow-lg transition-all duration-500 ease-out hover:bg-pos-0 hover:shadow-2xl">
      <a
        href={href}
        className="w-full no-underline leading-normal p-6 rounded-md text-white bg-[#23262d] opacity-95 transition-all duration-300 ease-in-out group-hover:bg-brand-yellow-soft group-hover:text-dark-200 group-hover:shadow-[0_18px_40px_rgba(250,164,11,0.25)]"
      >
        <h2 className="flex items-center gap-2 text-white transition-colors duration-300 ease-in-out group-hover:text-dark-200">
          {title}
          <span className="inline-block text-brand-pink transition-transform duration-300 ease-in-out group-hover:translate-x-1">
            &rarr;
          </span>
        </h2>
        <p className="mt-2 text-sm text-gray-300 group-hover:text-dark-200">{body}</p>
      </a>
    </li>
  );
}
