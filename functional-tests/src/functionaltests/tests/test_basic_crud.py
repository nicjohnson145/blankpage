from functionaltests import util
from blankpage.blankpage import v1 as blankpagev1
from betterproto2 import unwrap
from testfixtures import compare
from http import HTTPStatus


class TestBasicCrud(util.Base):

    def test_add(self):
        admin_token = self.login()

        # create a book
        resp = util.execute_http(
            util.blankpage_v1_method("AddBook"),
            key=admin_token,
            body=blankpagev1.AddBookRequest(
                book=blankpagev1.Book(
                    metadata=blankpagev1.Metadata(
                        title="Some Book",
                    ),
                    content=b"some-book-content",
                ),
            ),
            response_shape=blankpagev1.AddBookResponse,
        )

        # make sure we were assigned an ID
        self.assertNotEqual(unwrap(unwrap(resp.book).metadata).id, "")

    def test_list_basic(self):
        admin_token = self.login()

        # create a couple books
        for title in ["Book One", "Book Two", "Book Three"]:
            util.execute_http(
                util.blankpage_v1_method("AddBook"),
                key=admin_token,
                body=blankpagev1.AddBookRequest(
                    book=blankpagev1.Book(
                        metadata=blankpagev1.Metadata(
                            title=title,
                        ),
                        content=b"some-book-content",
                    ),
                ),
            )

        # do a basic list
        resp = util.execute_http(
            util.blankpage_v1_method("ListBooks"),
            key=admin_token,
            body=blankpagev1.ListBooksRequest(),
            response_shape=blankpagev1.ListBooksResponse,
        )

        # make sure we got them back in "most recent first" order
        compare(actual=[m.title for m in resp.books], expected=["Book Three", "Book Two", "Book One"])

    def test_read(self):
        admin_token = self.login()

        # create a book
        resp = util.execute_http(
            util.blankpage_v1_method("AddBook"),
            key=admin_token,
            body=blankpagev1.AddBookRequest(
                book=blankpagev1.Book(
                    metadata=blankpagev1.Metadata(
                        title="Some Book",
                    ),
                    content=b"some-book-content",
                ),
            ),
            response_shape=blankpagev1.AddBookResponse,
        )

        # read it back
        read_resp = util.execute_http(
            util.blankpage_v1_method("ReadBook"),
            key=admin_token,
            body=blankpagev1.ReadBookRequest(
                book_id=unwrap(unwrap(resp.book).metadata).id,
            ),
            response_shape=blankpagev1.ReadBookResponse,
        )
        # stip the uploadedAt field off, since testing that really sucks
        unwrap(read_resp.metadata).uploaded_at = None

        # everything else should match though
        compare(
            actual=read_resp.metadata,
            expected=blankpagev1.Metadata(
                id=unwrap(unwrap(resp.book).metadata).id,
                title="Some Book",
            ),
        )

    def test_update_field_mask(self):
        admin_token = self.login()

        # create a book
        resp = util.execute_http(
            util.blankpage_v1_method("AddBook"),
            key=admin_token,
            body=blankpagev1.AddBookRequest(
                book=blankpagev1.Book(
                    metadata=blankpagev1.Metadata(
                        title="Some Book",
                    ),
                    content=b"some-book-content",
                ),
            ),
            response_shape=blankpagev1.AddBookResponse,
        )

        # update that book, using a fieldmask

        # cant use FieldMask struct directly until https://github.com/betterproto/python-betterproto2/issues/181
        # is resolved
        body = blankpagev1.UpdateBookRequest(
            book=blankpagev1.Book(
                metadata=blankpagev1.Metadata(
                    id=unwrap(unwrap(resp.book).metadata).id,
                    author="Some Author",
                ),
            ),
        ).to_dict()
        body["field_mask"] = "metadata.author"
        util.execute_http(
            util.blankpage_v1_method("UpdateBook"),
            key=admin_token,
            body=body,
        )

        # read it back
        read_resp = util.execute_http(
            util.blankpage_v1_method("ReadBook"),
            key=admin_token,
            body=blankpagev1.ReadBookRequest(
                book_id=unwrap(unwrap(resp.book).metadata).id,
            ),
            response_shape=blankpagev1.ReadBookResponse,
        )
        # stip the uploadedAt field off, since testing that really sucks
        unwrap(read_resp.metadata).uploaded_at = None

        # everything else should match though
        compare(
            actual=read_resp.metadata,
            expected=blankpagev1.Metadata(
                id=unwrap(unwrap(resp.book).metadata).id, title="Some Book", author="Some Author"
            ),
        )

    def test_list_out_of_bounds(self):
        admin_token = self.login()

        # create a couple books
        for title in ["Book One", "Book Two", "Book Three"]:
            util.execute_http(
                util.blankpage_v1_method("AddBook"),
                key=admin_token,
                body=blankpagev1.AddBookRequest(
                    book=blankpagev1.Book(
                        metadata=blankpagev1.Metadata(
                            title=title,
                        ),
                        content=b"some-book-content",
                    ),
                ),
            )

        # ask for a page that doesnt exist
        resp = util.execute_http(
            util.blankpage_v1_method("ListBooks"),
            key=admin_token,
            body=blankpagev1.ListBooksRequest(
                pagination_options=blankpagev1.ListBooksRequestPaginationOptions(
                    per_page=25,
                    page=3,
                ),
            ),
            response_shape=blankpagev1.ListBooksResponse,
        )
        compare(actual=resp.books, expected=[])

    def test_list_multiple_pages(self):
        admin_token = self.login()

        # create a couple books
        for title in ["Book One", "Book Two", "Book Three"]:
            util.execute_http(
                util.blankpage_v1_method("AddBook"),
                key=admin_token,
                body=blankpagev1.AddBookRequest(
                    book=blankpagev1.Book(
                        metadata=blankpagev1.Metadata(
                            title=title,
                        ),
                        content=b"some-book-content",
                    ),
                ),
            )

        # ask for a page that doesnt exist
        resp = util.execute_http(
            util.blankpage_v1_method("ListBooks"),
            key=admin_token,
            body=blankpagev1.ListBooksRequest(
                pagination_options=blankpagev1.ListBooksRequestPaginationOptions(
                    per_page=2,
                ),
            ),
            response_shape=blankpagev1.ListBooksResponse,
        )

        # make sure we got back 2, but shows more available
        compare(actual=[m.title for m in resp.books], expected=["Book Three", "Book Two"])
        compare(actual=resp.has_more, expected=True)

    def test_remove_book(self):
        admin_token = self.login()

        # create a book
        resp = util.execute_http(
            util.blankpage_v1_method("AddBook"),
            key=admin_token,
            body=blankpagev1.AddBookRequest(
                book=blankpagev1.Book(
                    metadata=blankpagev1.Metadata(
                        title="Some Book",
                    ),
                    content=b"some-book-content",
                ),
            ),
            response_shape=blankpagev1.AddBookResponse,
        )

        # remove it
        util.execute_http(
            util.blankpage_v1_method("RemoveBook"),
            key=admin_token,
            body=blankpagev1.RemoveBookRequest(
                book_id=unwrap(unwrap(resp.book).metadata).id,
            ),
        )


class TestShelves(util.Base):
    def test_create_shelf(self):
        admin_token = self.login()

        # create a shelf
        resp = util.execute_http(
            util.blankpage_v1_method("CreateShelf"),
            key=admin_token,
            body=blankpagev1.CreateShelfRequest(
                shelf=blankpagev1.Shelf(
                    name="Some Shelf",
                ),
            ),
            response_shape=blankpagev1.CreateShelfResponse,
        )

        # make sure we were assigned an ID
        self.assertNotEqual(unwrap(resp.shelf).id, "")

    def test_list_shelves(self):
        admin_token = self.login()

        # create a couple shelves
        for name in ["Shelf One", "Shelf Two"]:
            util.execute_http(
                util.blankpage_v1_method("CreateShelf"),
                key=admin_token,
                body=blankpagev1.CreateShelfRequest(
                    shelf=blankpagev1.Shelf(
                        name=name,
                    ),
                ),
            )

        # list them out
        resp = util.execute_http(
            util.blankpage_v1_method("ListShelves"),
            key=admin_token,
            body=blankpagev1.ListShelvesRequest(),
            response_shape=blankpagev1.ListShelvesResponse,
        )
        compare(actual=[s.name for s in resp.shelves], expected=["Shelf Two", "Shelf One"])

    def test_add_to_shelf_successful(self):
        admin_token = self.login()

        # make a shelf
        shelf_resp = util.execute_http(
            util.blankpage_v1_method("CreateShelf"),
            key=admin_token,
            body=blankpagev1.CreateShelfRequest(
                shelf=blankpagev1.Shelf(
                    name="My Shelf",
                ),
            ),
            response_shape=blankpagev1.CreateShelfResponse,
        )

        # create a book
        book_resp = util.execute_http(
            util.blankpage_v1_method("AddBook"),
            key=admin_token,
            body=blankpagev1.AddBookRequest(
                book=blankpagev1.Book(
                    metadata=blankpagev1.Metadata(
                        title="Some Book",
                    ),
                    content=b"some-book-content",
                ),
            ),
            response_shape=blankpagev1.AddBookResponse,
        )

        # add the book to the shelf
        util.execute_http(
            util.blankpage_v1_method("AddBooksToShelf"),
            key=admin_token,
            body=blankpagev1.AddBooksToShelfRequest(
                shelf_id=unwrap(shelf_resp.shelf).id,
                book_ids=[
                    unwrap(unwrap(book_resp.book).metadata).id,
                ],
            ),
        )

    def test_add_to_shelf_bad_book(self):
        admin_token = self.login()

        # make a shelf
        shelf_resp = util.execute_http(
            util.blankpage_v1_method("CreateShelf"),
            key=admin_token,
            body=blankpagev1.CreateShelfRequest(
                shelf=blankpagev1.Shelf(
                    name="My Shelf",
                ),
            ),
            response_shape=blankpagev1.CreateShelfResponse,
        )

        # add a nonexistent book to the shelf
        util.execute_http(
            util.blankpage_v1_method("AddBooksToShelf"),
            key=admin_token,
            body=blankpagev1.AddBooksToShelfRequest(
                shelf_id=unwrap(shelf_resp.shelf).id,
                book_ids=["fake-id"],
            ),
            expected_status=HTTPStatus.BAD_REQUEST,
            error_contains="unknown book",
        )

    def test_add_to_shelf_bad_shelf(self):
        admin_token = self.login()

        # create a book
        book_resp = util.execute_http(
            util.blankpage_v1_method("AddBook"),
            key=admin_token,
            body=blankpagev1.AddBookRequest(
                book=blankpagev1.Book(
                    metadata=blankpagev1.Metadata(
                        title="Some Book",
                    ),
                    content=b"some-book-content",
                ),
            ),
            response_shape=blankpagev1.AddBookResponse,
        )

        # add a nonexistent book to the shelf
        util.execute_http(
            util.blankpage_v1_method("AddBooksToShelf"),
            key=admin_token,
            body=blankpagev1.AddBooksToShelfRequest(
                shelf_id="fake-shelf",
                book_ids=[unwrap(unwrap(book_resp.book).metadata).id],
            ),
            expected_status=HTTPStatus.BAD_REQUEST,
            error_contains="unknown shelf",
        )

    def test_remove_books_from_shelf(self):
        admin_token = self.login()

        # make a shelf
        shelf_resp = util.execute_http(
            util.blankpage_v1_method("CreateShelf"),
            key=admin_token,
            body=blankpagev1.CreateShelfRequest(
                shelf=blankpagev1.Shelf(
                    name="My Shelf",
                ),
            ),
            response_shape=blankpagev1.CreateShelfResponse,
        )

        # create a book
        book_resp = util.execute_http(
            util.blankpage_v1_method("AddBook"),
            key=admin_token,
            body=blankpagev1.AddBookRequest(
                book=blankpagev1.Book(
                    metadata=blankpagev1.Metadata(
                        title="Some Book",
                    ),
                    content=b"some-book-content",
                ),
            ),
            response_shape=blankpagev1.AddBookResponse,
        )

        # add the book to the shelf
        util.execute_http(
            util.blankpage_v1_method("AddBooksToShelf"),
            key=admin_token,
            body=blankpagev1.AddBooksToShelfRequest(
                shelf_id=unwrap(shelf_resp.shelf).id,
                book_ids=[
                    unwrap(unwrap(book_resp.book).metadata).id,
                ],
            ),
        )

        # remove the book from the shelf, should also ignore removing a book thats not on the shelf
        util.execute_http(
            util.blankpage_v1_method("RemoveBooksFromShelf"),
            key=admin_token,
            body=blankpagev1.RemoveBooksFromShelfRequest(
                shelf_id=unwrap(shelf_resp.shelf).id,
                book_ids=[
                    unwrap(unwrap(book_resp.book).metadata).id,
                    "other-made-up-id",
                ],
            ),
        )

        # should be gone
        list_resp = util.execute_http(
            util.blankpage_v1_method("ListBooksForShelf"),
            key=admin_token,
            body=blankpagev1.ListBooksForShelfRequest(
                shelf_id=unwrap(shelf_resp.shelf).id,
            ),
            response_shape=blankpagev1.ListBooksForShelfResponse,
        )
        compare(actual=[b.title for b in list_resp.books], expected=[])

    def test_list_books_for_shelf(self):
        admin_token = self.login()

        # make a shelf
        shelf_resp = util.execute_http(
            util.blankpage_v1_method("CreateShelf"),
            key=admin_token,
            body=blankpagev1.CreateShelfRequest(
                shelf=blankpagev1.Shelf(
                    name="My Shelf",
                ),
            ),
            response_shape=blankpagev1.CreateShelfResponse,
        )

        # create a book
        book_resp = util.execute_http(
            util.blankpage_v1_method("AddBook"),
            key=admin_token,
            body=blankpagev1.AddBookRequest(
                book=blankpagev1.Book(
                    metadata=blankpagev1.Metadata(
                        title="Some Book",
                    ),
                    content=b"some-book-content",
                ),
            ),
            response_shape=blankpagev1.AddBookResponse,
        )

        # add the book to the shelf
        util.execute_http(
            util.blankpage_v1_method("AddBooksToShelf"),
            key=admin_token,
            body=blankpagev1.AddBooksToShelfRequest(
                shelf_id=unwrap(shelf_resp.shelf).id,
                book_ids=[
                    unwrap(unwrap(book_resp.book).metadata).id,
                ],
            ),
        )

        # list them out
        list_resp = util.execute_http(
            util.blankpage_v1_method("ListBooksForShelf"),
            key=admin_token,
            body=blankpagev1.ListBooksForShelfRequest(
                shelf_id=unwrap(shelf_resp.shelf).id,
            ),
            response_shape=blankpagev1.ListBooksForShelfResponse,
        )
        compare(actual=[b.title for b in list_resp.books], expected=["Some Book"])

    def test_remove_book(self):
        admin_token = self.login()

        # make a shelf
        shelf_resp = util.execute_http(
            util.blankpage_v1_method("CreateShelf"),
            key=admin_token,
            body=blankpagev1.CreateShelfRequest(
                shelf=blankpagev1.Shelf(
                    name="My Shelf",
                ),
            ),
            response_shape=blankpagev1.CreateShelfResponse,
        )

        # create a book
        book_resp = util.execute_http(
            util.blankpage_v1_method("AddBook"),
            key=admin_token,
            body=blankpagev1.AddBookRequest(
                book=blankpagev1.Book(
                    metadata=blankpagev1.Metadata(
                        title="Some Book",
                    ),
                    content=b"some-book-content",
                ),
            ),
            response_shape=blankpagev1.AddBookResponse,
        )

        # add the book to the shelf
        util.execute_http(
            util.blankpage_v1_method("AddBooksToShelf"),
            key=admin_token,
            body=blankpagev1.AddBooksToShelfRequest(
                shelf_id=unwrap(shelf_resp.shelf).id,
                book_ids=[
                    unwrap(unwrap(book_resp.book).metadata).id,
                ],
            ),
        )

        # remove the book
        util.execute_http(
            util.blankpage_v1_method("RemoveBook"),
            key=admin_token,
            body=blankpagev1.RemoveBookRequest(
                book_id=unwrap(unwrap(book_resp.book).metadata).id,
            ),
        )

        # it should be gone from the shelf
        list_resp = util.execute_http(
            util.blankpage_v1_method("ListBooksForShelf"),
            key=admin_token,
            body=blankpagev1.ListBooksForShelfRequest(
                shelf_id=unwrap(shelf_resp.shelf).id,
            ),
            response_shape=blankpagev1.ListBooksForShelfResponse,
        )
        compare(actual=[b.title for b in list_resp.books], expected=[])
