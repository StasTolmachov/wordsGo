CREATE EXTENSION IF NOT EXISTS "pgcrypto";

create table dictionary (
    id uuid primary key default gen_random_uuid(),
    original text not null,
    translation text not null,
    transcription text
);

create table users (
    id UUID primary key default gen_random_uuid(),
    email varchar(100) not null unique,
    password_hash varchar(255) not null,
    first_name varchar(100) not null,
    last_name varchar(100) not null
);

create table user_progress (
    user_id UUID,
    word_id UUID,
    is_learned boolean,
    correct_streak integer,
    total_mistakes integer,
    last_seen timestamp,
    constraint fk_user foreign key (user_id) references users(id) on delete cascade,
    constraint fk_word foreign key (word_id) references dictionary(id) on delete cascade
);

create index idx_user_progress on user_progress(user_id, is_learned, correct_streak, last_seen);