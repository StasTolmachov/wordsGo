CREATE EXTENSION IF NOT EXISTS "pgcrypto";

create table dictionary (
    id uuid primary key default gen_random_uuid(),
    original text not null,
    translation text not null,
    transcription text,
    pos varchar(100),
    level varchar(100),
    past_simple_singular varchar(100),
    past_simple_plural varchar(100),
    past_participle_singular varchar(100),
    past_participle_plural varchar(100),
    Synonyms varchar(1000)
);

CREATE UNIQUE INDEX idx_dictionary_original ON dictionary (original);

create table users (
    id UUID primary key default gen_random_uuid(),
    email varchar(100) not null unique,
    password_hash varchar(255) not null,
    first_name varchar(100) not null,
    last_name varchar(100) not null,
    role varchar(20) not null default 'user',
    source_lang varchar(100) not null,
    target_lang varchar(100) not null,
    created_at TIMESTAMPTZ not null default now(),
    updated_at TIMESTAMPTZ not null default now(),
    deleted_at TIMESTAMPTZ null
);

alter table users
add constraint check_role check ( role in ('user', 'moderator', 'admin') );

create table user_progress (
    user_id UUID,
    word_id UUID,
    is_learned boolean,
    correct_streak integer,
    total_mistakes integer,
    DifficultyLevel float,
    last_seen timestamp,
    constraint fk_user foreign key (user_id) references users(id) on delete cascade,
    constraint fk_word foreign key (word_id) references dictionary(id) on delete cascade
);

create index idx_user_progress on user_progress(user_id, is_learned, correct_streak, last_seen);
CREATE UNIQUE INDEX idx_user_word_unique ON user_progress(user_id, word_id);