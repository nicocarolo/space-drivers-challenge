use space_drivers;

create table travels
(
    id      int auto_increment,
    user_id int         null,
    `from`  varchar(50) not null,
    `to`    varchar(50) not null,
    status  varchar(15) not null,
    constraint travel_id_uindex
        unique (id)
);

create index travels_status_index
    on travels (status);

create index travels_user_id_index
    on travels (user_id);

alter table travels
    add primary key (id);

create table users
(
    id       int auto_increment,
    email    varchar(50)  not null,
    password varchar(100) not null,
    role     varchar(10)  not null,
    constraint users_email_uindex
        unique (email),
    constraint users_id_uindex
        unique (id)
);

create index users_role_index
    on users (role);

alter table users
    add primary key (id);


-- create a first admin with password hola1234 to be able to create more users
INSERT INTO users (email, password, role) VALUES ('nico.carolo@hotmail.com', '$2a$10$0XNkz7egiyAPQbAEHvRtiOSIO/13.7ke0glVTZqkOC7gOl5BP6Ele', 'admin');
